package service

import (
	"context"
	"fmt"

	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math/big"
	"os"
	"strings"

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

	token, err := middleware.GenerateJWT(s.secret, user.ID, user.Role, user.AuthVersion)
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
	if err != nil || user.Email == nil || *user.Email == "" {
		return nil
	}
	token, err := generateRandomPassword(64)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(token))
	if err = s.userRepo.StoreReset(ctx, user.ID, hex.EncodeToString(hash[:])); err != nil {
		return err
	}
	base := strings.TrimRight(os.Getenv("WEB_URL"), "/")
	if base == "" {
		base = "http://localhost:3000"
	}
	event := model.NotificationEvent{EventID: uuid.NewString(), UserID: user.ID, Type: "PASSWORD_RESET", Metadata: map[string]string{"reset_url": base + "/reset-password?token=" + token}}
	if s.publisher == nil {
		return fmt.Errorf("email service unavailable")
	}
	if err = s.publisher.Publish(ctx, queue.NotificationQueue, event); err != nil {
		log.Printf("[AUTH] reset delivery failed: %v", err)
		return err
	}
	return nil
}
func (s *AuthService) ResetPassword(ctx context.Context, token, password string) error {
	if len(token) != 64 || len(password) < 8 || len(password) > 72 {
		return fmt.Errorf("mật khẩu phải có 8–72 ký tự; liên kết phải hợp lệ")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(token))
	return s.userRepo.ConsumeReset(ctx, hex.EncodeToString(sum[:]), string(hashed))
}

func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	if len(newPassword) < 8 || len(newPassword) > 72 {
		return fmt.Errorf("mật khẩu phải có 8–72 ký tự")
	}
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
