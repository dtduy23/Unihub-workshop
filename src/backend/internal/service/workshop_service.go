package service

import (
	"context"
	"fmt"
	"time"

	"unihub-workshop/internal/model"
	"unihub-workshop/internal/repository"
)

type WorkshopService struct {
	repo *repository.WorkshopRepo
}

func NewWorkshopService(repo *repository.WorkshopRepo) *WorkshopService {
	return &WorkshopService{repo: repo}
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
	return s.repo.FindByID(ctx, id)
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

	w := &model.Workshop{
		Title:          req.Title,
		Speaker:        &req.Speaker,
		Room:           req.Room,
		StartTime:      startTime,
		EndTime:        endTime,
		Capacity:       req.Capacity,
		AvailableSeats: req.Capacity,
		Price:          req.Price,
		Summary:        &req.Summary,
		RoomLayoutURL:  &req.RoomLayoutURL,
		Status:         model.WorkshopPublished,
	}

	if err := s.repo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("failed to create workshop: %w", err)
	}
	return w, nil
}

func (s *WorkshopService) Update(ctx context.Context, id string, req *model.UpdateWorkshopRequest) error {
	return s.repo.Update(ctx, id, req)
}

func (s *WorkshopService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
