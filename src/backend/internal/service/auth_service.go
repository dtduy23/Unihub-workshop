package service

import (
	"context"
	"fmt"

	"crypto/rand"
	"math/big"
	"log"

	"github.com/google/uuid"
	"unihub-workshop/internal/middleware"
	"unihub-workshop/internal/model"
	"unihub-workshop/internal/queue"
	"unihub-workshop/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo  *repository.UserRepo
	secret    string
	publisher *queue.Publisher
}

func NewAuthService(userRepo *repository.UserRepo, secret string, publisher *queue.Publisher) *AuthService {
	return &AuthService{userRepo: userRepo, secret: secret, publisher: publisher}
}

func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	user, err := s.userRepo.FindByStudentID(ctx, req.StudentID)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	token, err := middleware.GenerateJWT(s.secret, user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &model.LoginResponse{Token: token, User: *user}, nil
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*model.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

func generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, identifier string) error {
	user, err := s.userRepo.FindByStudentID(ctx, identifier)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Generate 32 char random password
	newPassword, err := generateRandomPassword(32)
	if err != nil {
		return fmt.Errorf("failed to generate random password: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update DB
	err = s.userRepo.UpdatePassword(ctx, user.ID, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Send notification if email exists
	if user.Email != nil && *user.Email != "" && s.publisher != nil {
		event := model.NotificationEvent{
			EventID:       uuid.New().String(),
			UserID:        user.ID,
			Type:          "FORGOT_PASSWORD",
			WorkshopTitle: "System",
			Metadata: map[string]string{
				"new_password": newPassword,
			},
		}

		err = s.publisher.Publish(ctx, queue.NotificationQueue, event)
		if err != nil {
			log.Printf("[AUTH] Failed to publish forgot password notification: %v", err)
			// Return nil anyway because password is changed successfully
		}
	}

	return nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Compare old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("incorrect old password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update DB
	err = s.userRepo.UpdatePassword(ctx, user.ID, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
