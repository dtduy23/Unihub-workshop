package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"unihub-workshop/internal/model"
	"unihub-workshop/internal/repository"

	"github.com/redis/go-redis/v9"
)

type WorkshopService struct {
	repo  *repository.WorkshopRepo
	redis *redis.Client
}

func NewWorkshopService(repo *repository.WorkshopRepo, redisClient *redis.Client) *WorkshopService {
	return &WorkshopService{
		repo:  repo,
		redis: redisClient,
	}
}

func (s *WorkshopService) ListAll(ctx context.Context, title string) ([]model.Workshop, error) {
	workshops, err := s.repo.FindAll(ctx, title)
	if err != nil {
		return nil, err
	}

	// GIẢ LẬP KIỂM TRA TẢI HỆ THỐNG
	// Trong thực tế, giá trị này có thể lấy từ Prometheus, Redis hoặc Metrics nội bộ
	currentRequestRate := 5000 // Giả sử hiện tại là 5.000 req/s
	MAX_ALLOWED_LOAD := 12000

	if currentRequestRate > MAX_ALLOWED_LOAD {
		// Nếu quá tải, Server sẽ ẩn toàn bộ Sơ đồ phòng để tiết kiệm tài nguyên
		for i := range workshops {
			workshops[i].RoomLayoutURL = nil
		}
	}

	return workshops, nil
}

func (s *WorkshopService) GetByID(ctx context.Context, id string) (*model.Workshop, error) {
	cacheKey := fmt.Sprintf("workshop:meta:%s", id)

	// 1. Check Redis cache first
	if s.redis != nil {
		if val, err := s.redis.Get(ctx, cacheKey).Result(); err == nil && val != "" {
			var w model.Workshop
			if err := json.Unmarshal([]byte(val), &w); err == nil {
				return &w, nil
			}
		}
	}

	// 2. Cache miss: Fetch from DB
	w, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Save to Redis cache for 10 minutes
	if s.redis != nil {
		if bytes, err := json.Marshal(w); err == nil {
			_ = s.redis.Set(ctx, cacheKey, bytes, 10*time.Minute).Err()
		}
	}

	return w, nil
}

func (s *WorkshopService) Create(ctx context.Context, req *model.CreateWorkshopRequest) (*model.Workshop, error) {
	return s.CreateFor(ctx, req, nil, nil)
}
func (s *WorkshopService) CreateFor(ctx context.Context, req *model.CreateWorkshopRequest, companyID, creator *string) (*model.Workshop, error) {
	if err := ValidateWorkshop(req); err != nil {
		return nil, err
	}
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time format: %w", err)
	}
	registrationStartTime, err := time.Parse(time.RFC3339, req.RegistrationStartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid registration_start_time format: %w", err)
	}
	registrationEndTime, err := time.Parse(time.RFC3339, req.RegistrationEndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid registration_end_time format: %w", err)
	}

	w := &model.Workshop{
		Title:       req.Title,
		Description: &req.Description, CompanyID: companyID, CreatedBy: creator,
		CoverURL: req.CoverURL, Audience: req.Audience, Benefits: req.Benefits, Preparation: req.Preparation, Agenda: req.Agenda, Format: req.Format,
		Speaker:               &req.Speaker,
		Room:                  req.Room,
		StartTime:             startTime,
		EndTime:               endTime,
		RegistrationStartTime: registrationStartTime,
		RegistrationEndTime:   registrationEndTime,
		Capacity:              req.Capacity,
		AvailableSeats:        req.Capacity,
		Price:                 req.Price,
		Summary:               &req.Summary,
		RoomLayoutURL:         &req.RoomLayoutURL,
		Status:                model.WorkshopPublished,
	}

	if companyID != nil {
		w.Status = model.WorkshopDraft
	}
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("failed to create workshop: %w", err)
	}
	return w, nil
}

func (s *WorkshopService) Update(ctx context.Context, id string, req *model.UpdateWorkshopRequest) error {
	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	merged := WorkshopRequest(current)
	if req.Title != nil {
		merged.Title = *req.Title
	}
	if req.Description != nil {
		merged.Description = *req.Description
	}
	if req.Speaker != nil {
		merged.Speaker = *req.Speaker
	}
	if req.Room != nil {
		merged.Room = *req.Room
	}
	if req.StartTime != nil {
		merged.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		merged.EndTime = *req.EndTime
	}
	if req.RegistrationStartTime != nil {
		merged.RegistrationStartTime = *req.RegistrationStartTime
	}
	if req.RegistrationEndTime != nil {
		merged.RegistrationEndTime = *req.RegistrationEndTime
	}
	if req.Capacity != nil {
		merged.Capacity = *req.Capacity
	}
	if req.Price != nil {
		merged.Price = *req.Price
	}
	if req.Format != nil {
		merged.Format = *req.Format
	}
	if err := ValidateWorkshop(&merged); err != nil {
		return err
	}
	if req.Status != nil {
		switch model.WorkshopStatus(*req.Status) {
		case model.WorkshopPublished, model.WorkshopClosed, model.WorkshopDeleted, model.WorkshopDraft, model.WorkshopPending, model.WorkshopRejected, model.WorkshopCancelled:
		default:
			return fmt.Errorf("invalid status")
		}
	}

	if err := s.repo.Update(ctx, id, req); err != nil {
		return err
	}

	// Invalidate cache
	if s.redis != nil {
		_ = s.redis.Del(ctx, fmt.Sprintf("workshop:meta:%s", id)).Err()
	}
	return nil
}

func (s *WorkshopService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	if s.redis != nil {
		_ = s.redis.Del(ctx, fmt.Sprintf("workshop:meta:%s", id)).Err()
	}
	return nil
}

func ValidateWorkshop(req *model.CreateWorkshopRequest) error {
	if strings.TrimSpace(req.Title) == "" || len(req.Title) > 200 || strings.TrimSpace(req.Room) == "" || strings.TrimSpace(req.Speaker) == "" {
		return fmt.Errorf("vui lòng nhập tiêu đề, diễn giả và địa điểm")
	}
	if req.Capacity < 1 || req.Capacity > 100000 || req.Price != 0 {
		return fmt.Errorf("sức chứa phải từ 1–100000; hiện chỉ hỗ trợ workshop miễn phí")
	}
	if req.Format == "" {
		req.Format = "OFFLINE"
	}
	if req.Format != "OFFLINE" && req.Format != "ONLINE" && req.Format != "HYBRID" {
		return fmt.Errorf("invalid format")
	}
	start, e1 := time.Parse(time.RFC3339, req.StartTime)
	end, e2 := time.Parse(time.RFC3339, req.EndTime)
	rs, e3 := time.Parse(time.RFC3339, req.RegistrationStartTime)
	re, e4 := time.Parse(time.RFC3339, req.RegistrationEndTime)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil || !start.Before(end) || !rs.Before(re) || re.After(start) {
		return fmt.Errorf("lịch workshop hoặc thời gian đăng ký không hợp lệ")
	}
	return nil
}
func WorkshopRequest(w *model.Workshop) model.CreateWorkshopRequest {
	req := model.CreateWorkshopRequest{Title: w.Title, Room: w.Room, StartTime: w.StartTime.Format(time.RFC3339), EndTime: w.EndTime.Format(time.RFC3339), RegistrationStartTime: w.RegistrationStartTime.Format(time.RFC3339), RegistrationEndTime: w.RegistrationEndTime.Format(time.RFC3339), Capacity: w.Capacity, Price: w.Price, CoverURL: w.CoverURL, Audience: w.Audience, Benefits: w.Benefits, Preparation: w.Preparation, Agenda: w.Agenda, Format: w.Format}
	if w.Description != nil {
		req.Description = *w.Description
	}
	if w.Speaker != nil {
		req.Speaker = *w.Speaker
	}
	if w.Summary != nil {
		req.Summary = *w.Summary
	}
	if w.RoomLayoutURL != nil {
		req.RoomLayoutURL = *w.RoomLayoutURL
	}
	return req
}

func (s *WorkshopService) ListFor(ctx context.Context, title string, admin bool) ([]model.Workshop, error) {
	return s.repo.List(ctx, title, admin, "")
}
func (s *WorkshopService) CompanyApproved(ctx context.Context, id string) bool {
	var ok bool
	_ = s.repo.GetPool().QueryRow(ctx, "SELECT status='APPROVED' FROM companies WHERE id=$1", id).Scan(&ok)
	return ok
}
func (s *WorkshopService) DeleteCache(ctx context.Context, id string) {
	if s.redis != nil {
		_ = s.redis.Del(ctx, "workshop:meta:"+id, "workshop:seats:"+id).Err()
	}
}
