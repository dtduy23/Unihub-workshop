package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"unihub-workshop/internal/crypto"
	"unihub-workshop/internal/model"
	"unihub-workshop/internal/queue"
	"unihub-workshop/internal/ratelimiter"
	"unihub-workshop/internal/repository"
)

type RegistrationService struct {
	regRepo        *repository.RegistrationRepo
	workshopRepo   *repository.WorkshopRepo
	userRepo       *repository.UserRepo
	paymentService *PaymentService
	crypto         *crypto.RSAProvider
	publisher    *queue.Publisher
	redis        *redis.Client
	waitingRoom  *ratelimiter.WaitingRoom
	seatLimiter  *ratelimiter.SeatLimiter
	gatewayURL   string
	mu           sync.RWMutex
	statuses     map[string]*model.RegistrationStatusResponse
}

func NewRegistrationService(
	regRepo *repository.RegistrationRepo,
	workshopRepo *repository.WorkshopRepo,
	userRepo *repository.UserRepo,
	paymentService *PaymentService,
	cryptoProvider *crypto.RSAProvider,
	publisher *queue.Publisher,
	redisClient *redis.Client,
	waitingRoom *ratelimiter.WaitingRoom,
	seatLimiter *ratelimiter.SeatLimiter,
) *RegistrationService {
	return &RegistrationService{
		regRepo:        regRepo,
		workshopRepo:   workshopRepo,
		userRepo:       userRepo,
		paymentService: paymentService,
		crypto:         cryptoProvider,
		publisher:      publisher,
		redis:          redisClient,
		waitingRoom:    waitingRoom,
		seatLimiter:    seatLimiter,
		statuses:       make(map[string]*model.RegistrationStatusResponse),
	}
}

// CheckWaitingRoom checks a user's status in the virtual waiting room.
// It also performs "lazy promotion" — each poll triggers a batch of queued users
// to be promoted into the active set, keeping the queue flowing.
func (s *RegistrationService) CheckWaitingRoom(ctx context.Context, workshopID, userID string) (*ratelimiter.WaitingRoomResult, error) {
	// Lazy promotion: promote waiting users before checking this user's status.
	// This ensures the queue keeps moving even without a dedicated background worker.
	if _, err := s.waitingRoom.PromoteNext(ctx, workshopID); err != nil {
		log.Printf("[WAITING_ROOM] Promotion error (non-fatal): %v", err)
	}

	return s.waitingRoom.Enter(ctx, workshopID, userID)
}

// EnqueueRegistration pushes registration request to RabbitMQ and returns a correlation ID
func (s *RegistrationService) EnqueueRegistration(ctx context.Context, userID, workshopID string) (string, error) {
	// Check if already registered
	existing, _ := s.regRepo.FindByUserAndWorkshop(ctx, userID, workshopID)
	if existing != nil && (existing.Status == model.RegSuccess || existing.Status == model.RegPendingPayment) {
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
	workshop, err := s.workshopRepo.FindByID(ctx, workshopID)
	if err != nil {
		return "", fmt.Errorf("workshop not found: %w", err)
	}

	// CHECK: Nếu là workshop có phí mà cổng thanh toán đang bảo trì -> Chặn luôn
	if workshop.Price > 0 && s.paymentService.IsGatewayDown(ctx) {
		return "", fmt.Errorf("cổng thanh toán đang bảo trì, vui lòng quay lại sau")
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
	workshop, err := s.workshopRepo.FindByID(ctx, msg.WorkshopID)
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
	var regStatus model.RegistrationStatus
	if workshop.Price > 0 {
		regStatus = model.RegPendingPayment
	} else {
		regStatus = model.RegSuccess
	}

	reg := &model.Registration{
		UserID:     msg.UserID,
		WorkshopID: msg.WorkshopID,
		Status:     regStatus,
	}

	if err := s.regRepo.Create(ctx, tx, reg); err != nil {
		tx.Rollback(ctx)
		
		// Xử lý lỗi trùng lặp (Idempotency)
		if strings.Contains(err.Error(), "uq_user_workshop") {
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
	if regStatus == model.RegSuccess && s.crypto != nil {
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

	// Nếu là workshop có phí, khởi tạo thanh toán ngay lập tức
	var paymentURL string
	var paymentAmount float64
	if regStatus == model.RegPendingPayment {
		pay, url, err := s.paymentService.InitiatePayment(ctx, reg.ID)
		if err != nil {
			log.Printf("[WORKER] Payment initiation failed: %v", err)
		} else {
			paymentURL = url
			paymentAmount = pay.Amount
		}
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
		Message:       fmt.Sprintf("Registration %s", regStatus),
		PaymentURL:    paymentURL,
		PaymentAmount: paymentAmount,
	})

	// If free workshop, publish notification event
	if regStatus == model.RegSuccess {
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
	}

	return nil
}

func (s *RegistrationService) GetStatus(correlationID string) *model.RegistrationStatusResponse {
	s.mu.RLock()
	status := s.statuses[correlationID]
	s.mu.RUnlock()

	// Nếu trạng thái là PENDING_PAYMENT, lấy thêm thông tin thanh toán
	if status != nil && status.Status == model.RegPendingPayment && status.Registration != nil {
		url, amount, err := s.paymentService.GetCheckoutURL(context.Background(), status.Registration.ID)
		if err == nil {
			status.PaymentURL = url
			status.PaymentAmount = amount
		}
	}

	return status
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
