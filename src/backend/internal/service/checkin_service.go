package service

import (
	"context"
	"log"
	"time"

	"unihub-workshop/internal/model"
	"unihub-workshop/internal/repository"
)

type CheckinService struct {
	regRepo *repository.RegistrationRepo
}

func NewCheckinService(regRepo *repository.RegistrationRepo) *CheckinService {
	return &CheckinService{regRepo: regRepo}
}

// LiveCheckin performs online check-in for a student
func (s *CheckinService) LiveCheckin(ctx context.Context, req *model.CheckinRequest) error {
	reg, err := s.regRepo.FindByStudentAndWorkshop(ctx, req.StudentID, req.WorkshopID)
	if err != nil {
		return err
	}
	if reg.IsCheckedIn {
		return nil // Already checked in, idempotent
	}
	return s.regRepo.CheckIn(ctx, reg.ID)
}

// BulkSync processes offline check-in records sent from mobile app
func (s *CheckinService) BulkSync(ctx context.Context, records []model.OfflineCheckinRecord) ([]string, []string) {
	var synced, failed []string

	for _, rec := range records {
		reg, err := s.regRepo.FindByStudentAndWorkshop(ctx, rec.StudentID, rec.WorkshopID)
		if err != nil {
			log.Printf("[CHECKIN_SYNC] Record %s failed - registration not found: %v", rec.ID, err)
			failed = append(failed, rec.ID)
			continue
		}

		// Conflict resolution: keep earliest timestamp
		if reg.IsCheckedIn {
			existingTime := reg.UpdatedAt.Unix()
			if rec.ScannedAt >= existingTime {
				log.Printf("[CHECKIN_SYNC] Record %s skipped - existing check-in is earlier", rec.ID)
				synced = append(synced, rec.ID) // Mark as synced since it's already checked in
				continue
			}
		}

		scannedAt := time.Unix(rec.ScannedAt, 0)
		_ = scannedAt
		if err := s.regRepo.CheckInWithTime(ctx, reg.ID, rec.ScannedAt); err != nil {
			log.Printf("[CHECKIN_SYNC] Record %s failed: %v", rec.ID, err)
			failed = append(failed, rec.ID)
			continue
		}

		synced = append(synced, rec.ID)
		log.Printf("[CHECKIN_SYNC] Record %s synced for student %s workshop %s", rec.ID, rec.StudentID, rec.WorkshopID)
	}

	return synced, failed
}
