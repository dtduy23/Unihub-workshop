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
	synced, failed := []string{}, []string{}

	for _, rec := range records {
		reg, err := s.regRepo.FindByStudentAndWorkshop(ctx, rec.StudentID, rec.WorkshopID)
		if err != nil {
			log.Printf("[CHECKIN_SYNC] Record %s failed - registration not found: %v", rec.ID, err)
			failed = append(failed, rec.ID)
			continue
		}

		if rec.ID == "" || rec.ScannedAt <= 0 || rec.ScannedAt > time.Now().Add(5*time.Minute).Unix() {
			failed = append(failed, rec.ID)
			continue
		}
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

func (s *CheckinService) Allowed(ctx context.Context, userID string, role model.Role, workshopID string) bool {
	if role == model.RoleAdmin {
		return true
	}
	var ok bool
	_ = s.regRepo.GetPool().QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM workshop_staff WHERE user_id=$1 AND workshop_id=$2)", userID, workshopID).Scan(&ok)
	return ok
}
