package service

import (
	"context"
	"encoding/json"
	"fmt"
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
		Title:                 req.Title,
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

	if err := s.repo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("failed to create workshop: %w", err)
	}
	return w, nil
}

func (s *WorkshopService) Update(ctx context.Context, id string, req *model.UpdateWorkshopRequest) error {
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
