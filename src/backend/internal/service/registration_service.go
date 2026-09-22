package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"unihub-workshop/internal/crypto"
	"unihub-workshop/internal/model"
	"unihub-workshop/internal/queue"
	"unihub-workshop/internal/repository"
	"unihub-workshop/internal/seatlimiter"
	"unihub-workshop/internal/waitingroom"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RegistrationService struct {
	regRepo       *repository.RegistrationRepo
	workshopRepo  *repository.WorkshopRepo
	userRepo      *repository.UserRepo
	crypto        *crypto.RSAProvider
	publisher     *queue.Publisher
	redis         *redis.Client
	waitingRoom   *waitingroom.WaitingRoom
	seatLimiter   *seatlimiter.SeatLimiter
	mu            sync.RWMutex
	statuses      map[string]*model.RegistrationStatusResponse
	lastPromoteMu sync.Mutex
	lastPromote   map[string]time.Time
}

func NewRegistrationService(
	regRepo *repository.RegistrationRepo,
	workshopRepo *repository.WorkshopRepo,
	userRepo *repository.UserRepo,
	cryptoProvider *crypto.RSAProvider,
	publisher *queue.Publisher,
	redisClient *redis.Client,
	waitingRoom *waitingroom.WaitingRoom,
	seatLimiter *seatlimiter.SeatLimiter,
) *RegistrationService {
	return &RegistrationService{
		regRepo:      regRepo,
		workshopRepo: workshopRepo,
		userRepo:     userRepo,
		crypto:       cryptoProvider,
		publisher:    publisher,
		redis:        redisClient,
		waitingRoom:  waitingRoom,
		seatLimiter:  seatLimiter,
		statuses:     make(map[string]*model.RegistrationStatusResponse),
		lastPromote:  make(map[string]time.Time),
	}
}

// GetCachedWorkshop retrieves workshop metadata from Redis if available,
// or falls back to DB and populates the Redis cache with a TTL of 10 minutes.
func (s *RegistrationService) GetCachedWorkshop(ctx context.Context, workshopID string) (*model.Workshop, error) {
	cacheKey := fmt.Sprintf("workshop:meta:%s", workshopID)

	// 1. Check Redis cache first (sub-millisecond in-memory read)
	if val, err := s.redis.Get(ctx, cacheKey).Result(); err == nil && val != "" {
		var w model.Workshop
		if err := json.Unmarshal([]byte(val), &w); err == nil {
			return &w, nil
		}
	}

	// 2. Cache miss: Fetch from Database (PostgreSQL)
	workshop, err := s.workshopRepo.FindByID(ctx, workshopID)
	if err != nil {
		return nil, err
	}

	// 3. Cache into Redis for 10 minutes to protect DB from thundering herd
	if bytes, err := json.Marshal(workshop); err == nil {
		_ = s.redis.Set(ctx, cacheKey, bytes, 10*time.Minute).Err()
	}

	return workshop, nil
}

// CheckWaitingRoom checks a user's status in the virtual waiting room.
// It also performs "lazy promotion" — each poll triggers a batch of queued users
// to be promoted into the active set, keeping the queue flowing.
func (s *RegistrationService) CheckWaitingRoom(ctx context.Context, workshopID, userID string) (*waitingroom.WaitingRoomResult, error) {
	workshop, err := s.GetCachedWorkshop(ctx, workshopID)
	if err != nil {
		return nil, fmt.Errorf("workshop not found: %w", err)
	}

	now := time.Now()
	if !workshop.RegistrationStartTime.IsZero() && now.Before(workshop.RegistrationStartTime) {
		return nil, fmt.Errorf("cổng đăng ký chưa mở (thời gian mở: %s)", workshop.RegistrationStartTime.Local().Format("15:04:05 02/01/2006"))
	}
	if !workshop.RegistrationEndTime.IsZero() && now.After(workshop.RegistrationEndTime) {
		return nil, fmt.Errorf("thời hạn đăng ký chuyên đề này đã kết thúc")
	}

	// Dynamic maxActive: use workshop.Capacity (fallback to 100 if <= 0)
	maxActive := workshop.Capacity
	if maxActive <= 0 {
		maxActive = 100
	}

	// Throttled asynchronous promotion (at most once every 250ms) to avoid saturating Redis Lua engine
	s.maybePromote(workshopID, maxActive)

	return s.waitingRoom.Enter(ctx, workshopID, userID, workshop.RegistrationStartTime, maxActive)
}

func (s *RegistrationService) maybePromote(workshopID string, maxActive int) {
	s.lastPromoteMu.Lock()
	last := s.lastPromote[workshopID]
	now := time.Now()
	if now.Sub(last) < 250*time.Millisecond {
		s.lastPromoteMu.Unlock()
		return
	}
	s.lastPromote[workshopID] = now
	s.lastPromoteMu.Unlock()

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := s.waitingRoom.PromoteNext(bgCtx, workshopID, maxActive); err != nil {
			log.Printf("[WAITING_ROOM] Promotion error (non-fatal): %v", err)
		}
	}()
}

// EnqueueRegistration pushes registration request to RabbitMQ and returns a correlation ID
func (s *RegistrationService) EnqueueRegistration(ctx context.Context, userID, workshopID string) (string, error) {
	// Check if already registered
	existing, _ := s.regRepo.FindByUserAndWorkshop(ctx, userID, workshopID)
	if existing != nil && existing.Status == model.RegSuccess {
		return "", fmt.Errorf("already registered for this workshop")
	}

	correlationID := uuid.New().String()

	msg := model.QueueMessage{
		CorrelationID: correlationID,
		UserID:        userID,
		WorkshopID:    workshopID,
		Action:        "REGISTER",
	}

	// Set initial status
	s.SetStatus(correlationID, &model.RegistrationStatusResponse{
		CorrelationID: correlationID,
		Status:        "PROCESSING",
		Message:       "Your registration is being processed",
	})

	// ==========================================
	// 2. REDIS SEAT LOCK (DOUBLE-CHECK)
	// ==========================================
	// Before enqueuing, we try to decrement the seat count in Redis.
	// This acts as a high-performance shield for the database.

	// Pre-warm cache if needed (Get workshop to know initial seats)
	workshop, err := s.GetCachedWorkshop(ctx, workshopID)
	if err != nil {
		return "", fmt.Errorf("workshop not found: %w", err)
	}

	// CHECK: Thời gian mở và đóng cổng đăng ký (Chống cURL / Bot lén lút)
	now := time.Now()
	if !workshop.RegistrationStartTime.IsZero() && now.Before(workshop.RegistrationStartTime) {
		return "", fmt.Errorf("cổng đăng ký chưa mở (thời gian mở: %s)",
			workshop.RegistrationStartTime.Local().Format("15:04:05 02/01/2006"))
	}
	if !workshop.RegistrationEndTime.IsZero() && now.After(workshop.RegistrationEndTime) {
		return "", fmt.Errorf("thời hạn đăng ký chuyên đề này đã kết thúc")
	}

	if err := s.seatLimiter.PrepareCache(ctx, workshopID, workshop.AvailableSeats); err != nil {
		return "", fmt.Errorf("failed to prepare seat cache: %w", err)
	}

	success, err := s.seatLimiter.TryAcquireSeat(ctx, workshopID)
	if err != nil {
		return "", fmt.Errorf("failed to acquire seat in Redis: %w", err)
	}

	if !success {
		return "", fmt.Errorf("workshop is full (verified by cache)")
	}

	if err := s.publisher.Publish(ctx, queue.RegistrationQueue, msg); err != nil {
		// Rollback Redis seat if publishing fails
		_ = s.seatLimiter.ReleaseSeat(ctx, workshopID)

		s.SetStatus(correlationID, &model.RegistrationStatusResponse{
			CorrelationID: correlationID,
			Status:        model.RegFailed,
			Message:       "System is temporarily unavailable",
		})
		return "", fmt.Errorf("failed to enqueue registration: %w", err)
	}

	log.Printf("[REGISTRATION] Enqueued: correlation=%s user=%s workshop=%s", correlationID, userID, workshopID)

	// Khởi tạo trạng thái PROCESSING ngay khi Enqueue thành công để tránh lỗi 404 ở Client
	s.SetStatus(correlationID, &model.RegistrationStatusResponse{
		CorrelationID: correlationID,
		Status:        model.RegProcessing,
		Message:       "Đang chờ xử lý trong hàng đợi...",
	})

	return correlationID, nil
}

// ProcessRegistration is called by the background worker to process a registration message
func (s *RegistrationService) ProcessRegistration(ctx context.Context, msg model.QueueMessage) error {
	log.Printf("[WORKER] Processing registration: correlation=%s", msg.CorrelationID)

	// Get workshop info to determine if it's free or paid
	workshop, err := s.GetCachedWorkshop(ctx, msg.WorkshopID)
	if err != nil {
		s.SetStatus(msg.CorrelationID, &model.RegistrationStatusResponse{
			CorrelationID: msg.CorrelationID,
			Status:        model.RegFailed,
			Message:       "Workshop not found",
		})
		return err
	}

	// Begin transaction with Pessimistic Locking
	tx, err := s.workshopRepo.GetPool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// SELECT FOR UPDATE - Pessimistic Lock
	_, err = s.workshopRepo.DecrementSeatWithLock(ctx, tx, msg.WorkshopID)
	if err != nil {
		// DB says no seats! (Inconsistency with Redis)
		// We should release the tentative seat we took in Redis
		_ = s.seatLimiter.ReleaseSeat(ctx, msg.WorkshopID)

		s.SetStatus(msg.CorrelationID, &model.RegistrationStatusResponse{
			CorrelationID: msg.CorrelationID,
			Status:        model.RegRejected,
			Message:       "No available seats (Database verified)",
		})
		return err
	}

	// Determine status based on price
	regStatus := model.RegSuccess

	reg := &model.Registration{
		UserID:     msg.UserID,
		WorkshopID: msg.WorkshopID,
		Status:     regStatus,
	}

	if err := s.regRepo.Create(ctx, tx, reg); err != nil {
		tx.Rollback(ctx)

		// Xử lý lỗi trùng lặp (Idempotency hoặc do UPSERT bị loại trừ)
		if strings.Contains(err.Error(), "uq_user_workshop") || strings.Contains(err.Error(), "no rows in result set") {
			log.Printf("[WORKER] Duplicate registration attempt: user=%s workshop=%s", msg.UserID, msg.WorkshopID)
			s.SetStatus(msg.CorrelationID, &model.RegistrationStatusResponse{
				CorrelationID: msg.CorrelationID,
				Status:        model.RegFailed,
				Message:       "Bạn đã đăng ký Workshop này rồi.",
			})
			return nil // Trả về nil để ACK tin nhắn, không retry nữa
		}

		s.SetStatus(msg.CorrelationID, &model.RegistrationStatusResponse{
			CorrelationID: msg.CorrelationID,
			Status:        model.RegFailed,
			Message:       "Lỗi hệ thống khi lưu bản ghi đăng ký",
		})
		return fmt.Errorf("failed to create registration: %w", err)
	}

	// Generate RSA Signature for successful registrations (now we have reg.ID)
	if s.crypto != nil {
		user, err := s.userRepo.FindByID(ctx, msg.UserID)
		if err == nil {
			// Sign with 4-field context for mobile: sid, uid, wid
			qrData, err := s.crypto.SignTicket(user.StudentID, msg.UserID, msg.WorkshopID)
			if err == nil {
				reg.TicketSignature = &qrData
				// Update the signature in DB
				_, _ = tx.Exec(ctx, "UPDATE registrations SET ticket_signature = $1 WHERE id = $2", qrData, reg.ID)
			} else {
				log.Printf("[WORKER] RSA signing failed: %v", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[WORKER] Registration finalized: id=%s status=%s", reg.ID, regStatus)

	// Release waiting room slot so next queued user can be promoted
	if err := s.waitingRoom.ReleaseAccess(ctx, msg.WorkshopID, msg.UserID); err != nil {
		log.Printf("[WAITING_ROOM] Failed to release access (non-fatal): %v", err)
	}

	s.SetStatus(msg.CorrelationID, &model.RegistrationStatusResponse{
		CorrelationID: msg.CorrelationID,
		Status:        regStatus,
		Registration:  reg,
		Message:       "Registration SUCCESS",
	})

	// Publish notification event
	notifEvent := model.NotificationEvent{
		EventID:         fmt.Sprintf("REG_SUCCESS_%s", reg.ID),
		UserID:          msg.UserID,
		RegistrationID:  reg.ID,
		Type:            "REGISTRATION_SUCCESS",
		WorkshopTitle:   workshop.Title,
		TicketSignature: "",
	}
	if reg.TicketSignature != nil {
		notifEvent.TicketSignature = *reg.TicketSignature
	}
	_ = s.publisher.Publish(ctx, queue.NotificationQueue, notifEvent)

	return nil
}

func (s *RegistrationService) GetStatus(correlationID string) *model.RegistrationStatusResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statuses[correlationID]
}

func (s *RegistrationService) SetStatus(correlationID string, status *model.RegistrationStatusResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses[correlationID] = status
}

func (s *RegistrationService) GetUserRegistrations(ctx context.Context, userID string) ([]model.Registration, error) {
	return s.regRepo.FindByUser(ctx, userID)
}
func (s *RegistrationService) GetUserRegistrationsWithWorkshop(ctx context.Context, userID string) ([]model.RegistrationWithWorkshop, error) {
	return s.regRepo.FindByUserWithWorkshop(ctx, userID)
}

func (s *RegistrationService) GetByWorkshop(ctx context.Context, workshopID string) ([]model.RegistrationWithUser, error) {
	return s.regRepo.FindByWorkshopWithUser(ctx, workshopID)
}

// CancelRegistration cancels an existing registration and frees up the seat
func (s *RegistrationService) CancelRegistration(ctx context.Context, userID, registrationID string) error {
	// 1. Fetch registration
	reg, err := s.regRepo.FindByID(ctx, registrationID)
	if err != nil {
		return fmt.Errorf("registration not found: %w", err)
	}

	// 2. Validate ownership
	if reg.UserID != userID {
		return fmt.Errorf("unauthorized to cancel this registration")
	}

	// 3. Validate status
	if reg.Status != model.RegSuccess {
		return fmt.Errorf("only SUCCESS registrations can be cancelled")
	}

	// 4. Update status in DB
	err = s.regRepo.UpdateStatus(ctx, registrationID, model.RegCancelled)
	if err != nil {
		return fmt.Errorf("failed to cancel registration: %w", err)
	}

	// 5. Increment available seats in DB
	err = s.workshopRepo.IncrementSeat(ctx, reg.WorkshopID)
	if err != nil {
		log.Printf("[ERROR] Failed to increment seat in DB for workshop %s after cancellation: %v", reg.WorkshopID, err)
		// We log but don't fail the cancellation if DB increment fails, although this is rare.
	}

	// 6. Increment available seats in Redis Cache
	err = s.seatLimiter.ReleaseSeat(ctx, reg.WorkshopID)
	if err != nil {
		log.Printf("[ERROR] Failed to release seat in Redis for workshop %s: %v", reg.WorkshopID, err)
	}

	log.Printf("[REGISTRATION] Cancelled: registration=%s user=%s workshop=%s", registrationID, userID, reg.WorkshopID)

	return nil
}

// ExportCSV generates a CSV file of registrations for a workshop
func (s *RegistrationService) ExportCSV(ctx context.Context, workshopID string, exportType string) ([]byte, error) {
	regs, err := s.GetByWorkshop(ctx, workshopID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registrations: %w", err)
	}

	var buf bytes.Buffer
	// Write UTF-8 BOM so Excel reads it correctly
	buf.Write([]byte("\xEF\xBB\xBF"))

	cw := csv.NewWriter(&buf)

	// Headers
	_ = cw.Write([]string{"STT", "Mã sinh viên", "Họ và tên", "Email", "Trạng thái", "Đã điểm danh", "Ngày đăng ký"})

	stt := 1
	for _, reg := range regs {
		// Filter based on type
		if exportType == "attended" {
			if reg.Status != model.RegSuccess || !reg.IsCheckedIn {
				continue
			}
		} else { // "registered" or default
			if reg.Status != model.RegSuccess {
				continue
			}
		}

		var checkedIn string
		if reg.IsCheckedIn {
			checkedIn = "Có"
		} else {
			checkedIn = "Không"
		}

		dateStr := reg.CreatedAt.Local().Format("02/01/2006 15:04:05")

		_ = cw.Write([]string{
			fmt.Sprintf("%d", stt),
			reg.StudentID,
			reg.FullName,
			reg.Email,
			string(reg.Status),
			checkedIn,
			dateStr,
		})
		stt++
	}
	cw.Flush()

	if err := cw.Error(); err != nil {
		return nil, fmt.Errorf("error writing csv: %w", err)
	}

	return buf.Bytes(), nil
}
