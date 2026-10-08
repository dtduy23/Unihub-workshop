package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"log"
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
	workshop, err := s.workshopRepo.FindByID(ctx, workshopID)
	if err != nil {
		return nil, fmt.Errorf("workshop not found: %w", err)
	}

	if workshop.Status != model.WorkshopPublished {
		return nil, fmt.Errorf("workshop không nhận đăng ký")
	}
	now := time.Now()
	if !workshop.RegistrationStartTime.IsZero() && now.Before(workshop.RegistrationStartTime) {
		return nil, fmt.Errorf("cổng đăng ký chưa mở (thời gian mở: %s)", workshop.RegistrationStartTime.Local().Format("15:04:05 02/01/2006"))
	}
	if !workshop.RegistrationEndTime.IsZero() && now.After(workshop.RegistrationEndTime) {
		return nil, fmt.Errorf("thời hạn đăng ký chuyên đề này đã kết thúc")
	}

	// Dynamic maxActive: use workshop.Capacity (fallback to 100 if <= 0)
	if s.waitingRoom == nil {
		return &waitingroom.WaitingRoomResult{Status: waitingroom.QueueGranted}, nil
	}
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

// Seats are reserved durably before enqueue. SQL is authoritative; Redis is only metadata caching.
func (s *RegistrationService) EnqueueRegistration(ctx context.Context, userID, workshopID string) (string, error) {
	tx, err := s.workshopRepo.GetPool().Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var status string
	var seats int
	var start, end time.Time
	var approved bool
	err = tx.QueryRow(ctx, `SELECT w.status,w.available_seats,w.registration_start_time,w.registration_end_time,(w.company_id IS NULL OR c.status='APPROVED') FROM workshops w LEFT JOIN companies c ON c.id=w.company_id WHERE w.id=$1 FOR UPDATE OF w`, workshopID).Scan(&status, &seats, &start, &end, &approved)
	if err != nil {
		return "", fmt.Errorf("workshop not found")
	}
	if status != "PUBLISHED" || !approved || time.Now().Before(start) || time.Now().After(end) {
		return "", fmt.Errorf("workshop chưa mở hoặc đã đóng đăng ký")
	}
	var existing string
	err = tx.QueryRow(ctx, "SELECT id FROM registration_requests WHERE user_id=$1 AND workshop_id=$2 AND status='PROCESSING'", userID, workshopID).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if err != pgx.ErrNoRows {
		return "", err
	}
	var registered bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM registrations WHERE user_id=$1 AND workshop_id=$2 AND status='SUCCESS')", userID, workshopID).Scan(&registered); err != nil {
		return "", err
	}
	if registered {
		return "", fmt.Errorf("Bạn đã đăng ký workshop này")
	}
	if seats <= 0 {
		return "", fmt.Errorf("Workshop đã đủ chỗ")
	}
	id := uuid.NewString()
	if _, err = tx.Exec(ctx, "UPDATE workshops SET available_seats=available_seats-1 WHERE id=$1", workshopID); err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO registration_requests(id,user_id,workshop_id) VALUES($1,$2,$3)", id, userID, workshopID); err != nil {
		return "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	s.invalidateWorkshop(ctx, workshopID)
	msg := model.QueueMessage{CorrelationID: id, UserID: userID, WorkshopID: workshopID, Action: "REGISTER"}
	if s.publisher != nil {
		if err = s.publisher.Publish(ctx, queue.RegistrationQueue, msg); err == nil {
			_, _ = s.workshopRepo.GetPool().Exec(ctx, "UPDATE registration_requests SET published=true WHERE id=$1", id)
		}
	}
	// The outbox republishes unpublished reservations after crashes or broker failures.
	return id, nil
}
func (s *RegistrationService) ProcessRegistration(ctx context.Context, msg model.QueueMessage) error {
	var attempts int
	err := s.workshopRepo.GetPool().QueryRow(ctx, "UPDATE registration_requests SET attempts=attempts+1 WHERE id=$1 AND user_id=$2 AND workshop_id=$3 AND status='PROCESSING' RETURNING attempts", msg.CorrelationID, msg.UserID, msg.WorkshopID).Scan(&attempts)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if attempts > 3 {
		return s.finishFailed(ctx, msg, "Đã hết số lần xử lý")
	}
	var signature *string
	if s.crypto != nil {
		sid := s.getCachedStudentID(ctx, msg.UserID)
		qr, e := s.crypto.SignTicket(sid, msg.UserID, msg.WorkshopID)
		if e != nil {
			return e
		}
		signature = &qr
	}
	tx, err := s.workshopRepo.GetPool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var workshopStatus, title string
	var approved bool
	if err = tx.QueryRow(ctx, `SELECT w.status,w.title,(w.company_id IS NULL OR c.status='APPROVED') FROM workshops w LEFT JOIN companies c ON c.id=w.company_id WHERE w.id=$1 FOR UPDATE OF w`, msg.WorkshopID).Scan(&workshopStatus, &title, &approved); err != nil {
		return err
	}
	var state, userID, workshopID string
	if err = tx.QueryRow(ctx, "SELECT status,user_id,workshop_id FROM registration_requests WHERE id=$1 FOR UPDATE", msg.CorrelationID).Scan(&state, &userID, &workshopID); err != nil {
		return err
	}
	if state != "PROCESSING" {
		return nil
	}
	if userID != msg.UserID || workshopID != msg.WorkshopID {
		return fmt.Errorf("invalid queue identity")
	}
	if workshopStatus != "PUBLISHED" || !approved {
		tx.Rollback(ctx)
		return s.finishFailed(ctx, msg, "Workshop đã đóng hoặc doanh nghiệp ngừng hoạt động")
	}
	reg := &model.Registration{UserID: userID, WorkshopID: workshopID, Status: model.RegSuccess, TicketSignature: signature}
	if err = s.regRepo.Create(ctx, tx, reg); err != nil {
		tx.Rollback(ctx)
		if err == pgx.ErrNoRows {
			return s.finishFailed(ctx, msg, "Bạn đã có vé workshop này")
		}
		return err
	}
	if _, err = tx.Exec(ctx, "UPDATE registration_requests SET status='SUCCESS',registration_id=$1,message='Đăng ký thành công',updated_at=now() WHERE id=$2", reg.ID, msg.CorrelationID); err != nil {
		return err
	}
	// In-app notification is committed with the ticket, independently of email delivery.
	if _, err = tx.Exec(ctx, `INSERT INTO notifications(user_id,registration_id,channel,title,content,status,event_id,link) VALUES($1,$2,'WEB',$3,$4,'SENT',$5,'/') ON CONFLICT(event_id,channel) DO NOTHING`, userID, reg.ID, "Đăng ký thành công: "+title, "Vé workshop của bạn đã sẵn sàng.", "REG_SUCCESS_"+msg.CorrelationID); err != nil {
		return err
	}
	event := model.NotificationEvent{EventID: "REG_SUCCESS_" + msg.CorrelationID, UserID: userID, RegistrationID: reg.ID, WorkshopTitle: title, Type: "REGISTRATION_SUCCESS", Metadata: map[string]string{"email_only": "true"}}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO notification_outbox(event_id,payload) VALUES($1,$2) ON CONFLICT DO NOTHING", event.EventID, data); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	s.invalidateWorkshop(ctx, workshopID)
	s.releaseWaiting(ctx, workshopID, userID)
	return nil
}
func (s *RegistrationService) finishFailed(ctx context.Context, msg model.QueueMessage, reason string) error {
	tx, err := s.workshopRepo.GetPool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT id FROM workshops WHERE id=$1 FOR UPDATE", msg.WorkshopID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, "UPDATE registration_requests SET status='FAILED',message=$1,updated_at=now() WHERE id=$2 AND user_id=$3 AND workshop_id=$4 AND status='PROCESSING'", reason, msg.CorrelationID, msg.UserID, msg.WorkshopID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 1 {
		if _, err = tx.Exec(ctx, "UPDATE workshops SET available_seats=LEAST(available_seats+1,capacity) WHERE id=$1", msg.WorkshopID); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	s.invalidateWorkshop(ctx, msg.WorkshopID)
	s.releaseWaiting(ctx, msg.WorkshopID, msg.UserID)
	return nil
}
func (s *RegistrationService) FailAfterRetries(ctx context.Context, msg model.QueueMessage) bool {
	var attempts int
	if s.workshopRepo.GetPool().QueryRow(ctx, "SELECT attempts FROM registration_requests WHERE id=$1", msg.CorrelationID).Scan(&attempts) != nil {
		return false
	}
	return attempts >= 3 && s.finishFailed(ctx, msg, "Không thể hoàn tất đăng ký sau 3 lần xử lý") == nil
}
func (s *RegistrationService) PublishPending(ctx context.Context) {
	if s.publisher == nil {
		return
	}
	rows, err := s.workshopRepo.GetPool().Query(ctx, "SELECT id,user_id,workshop_id FROM registration_requests WHERE NOT published AND status='PROCESSING' ORDER BY created_at LIMIT 50")
	if err != nil {
		return
	}
	messages := []model.QueueMessage{}
	for rows.Next() {
		var m model.QueueMessage
		if rows.Scan(&m.CorrelationID, &m.UserID, &m.WorkshopID) == nil {
			m.Action = "REGISTER"
			messages = append(messages, m)
		}
	}
	rows.Close()
	for _, msg := range messages {
		if s.publisher.Publish(ctx, queue.RegistrationQueue, msg) == nil {
			_, _ = s.workshopRepo.GetPool().Exec(ctx, "UPDATE registration_requests SET published=true WHERE id=$1", msg.CorrelationID)
		}
	}
	emailRows, err := s.workshopRepo.GetPool().Query(ctx, "SELECT event_id,payload FROM notification_outbox WHERE NOT published ORDER BY created_at LIMIT 50")
	if err != nil {
		return
	}
	events := []model.NotificationEvent{}
	for emailRows.Next() {
		var id string
		var data []byte
		if emailRows.Scan(&id, &data) == nil {
			var event model.NotificationEvent
			if json.Unmarshal(data, &event) == nil {
				events = append(events, event)
			}
		}
	}
	emailRows.Close()
	for _, event := range events {
		if s.publisher.Publish(ctx, queue.NotificationQueue, event) == nil {
			_, _ = s.workshopRepo.GetPool().Exec(ctx, "UPDATE notification_outbox SET published=true WHERE event_id=$1", event.EventID)
		}
	}

}
func (s *RegistrationService) GetStatus(ctx context.Context, userID, id string) (*model.RegistrationStatusResponse, error) {
	response := &model.RegistrationStatusResponse{CorrelationID: id}
	var registrationID *string
	err := s.workshopRepo.GetPool().QueryRow(ctx, "SELECT status,message,registration_id FROM registration_requests WHERE id=$1 AND user_id=$2", id, userID).Scan(&response.Status, &response.Message, &registrationID)
	if err != nil {
		return nil, err
	}
	if registrationID != nil {
		response.Registration, err = s.regRepo.FindByID(ctx, *registrationID)
	}
	return response, err
}
func (s *RegistrationService) invalidateWorkshop(ctx context.Context, id string) {
	if s.redis != nil {
		_ = s.redis.Del(ctx, "workshop:meta:"+id, "workshop:seats:"+id).Err()
	}
}
func (s *RegistrationService) releaseWaiting(ctx context.Context, workshopID, userID string) {
	if s.waitingRoom != nil {
		_ = s.waitingRoom.ReleaseAccess(ctx, workshopID, userID)
	}
}

// getCachedStudentID retrieves the user's studentID with 24h Redis caching
func (s *RegistrationService) getCachedStudentID(ctx context.Context, userID string) string {
	cacheKey := fmt.Sprintf("user:sid:%s", userID)
	if s.redis != nil {
		if sid, err := s.redis.Get(ctx, cacheKey).Result(); err == nil && sid != "" {
			return sid
		}
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Printf("[WORKER] User not found for studentID lookup: %v", err)
		return ""
	}
	if s.redis != nil {
		_ = s.redis.Set(ctx, cacheKey, user.StudentID, 24*time.Hour).Err()
	}
	return user.StudentID
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
	reg, err := s.regRepo.FindByID(ctx, registrationID)
	if err != nil {
		return err
	}
	if reg.UserID != userID {
		return fmt.Errorf("không có quyền hủy vé")
	}
	tx, err := s.workshopRepo.GetPool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT id FROM workshops WHERE id=$1 FOR UPDATE", reg.WorkshopID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, "UPDATE registrations SET status='CANCELLED',updated_at=now() WHERE id=$1 AND user_id=$2 AND status='SUCCESS' AND NOT is_checked_in", registrationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var state string
		var checked bool
		if err = tx.QueryRow(ctx, "SELECT status,is_checked_in FROM registrations WHERE id=$1", registrationID).Scan(&state, &checked); err != nil {
			return err
		}
		if state == "CANCELLED" {
			return nil
		}
		return fmt.Errorf("không thể hủy vé đã sử dụng hoặc không còn hiệu lực")
	}
	if _, err = tx.Exec(ctx, "UPDATE workshops SET available_seats=LEAST(available_seats+1,capacity) WHERE id=$1", reg.WorkshopID); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	s.invalidateWorkshop(ctx, reg.WorkshopID)
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
