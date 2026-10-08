package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"html"
	"log"
	"net/smtp"
	"strings"

	"unihub-workshop/internal/model"
	"unihub-workshop/internal/repository"
)

// NotificationStrategy interface (Strategy Pattern)
type NotificationStrategy interface {
	Send(ctx context.Context, notif *model.Notification) error
	Channel() model.NotificationChannel
}

// NotificationService dispatches notifications using multiple strategies (Observer + Strategy Pattern)
type NotificationService struct {
	repo       *repository.NotificationRepo
	strategies []NotificationStrategy
}

func NewNotificationService(repo *repository.NotificationRepo, strategies ...NotificationStrategy) *NotificationService {
	return &NotificationService{repo: repo, strategies: strategies}
}

// Dispatch keeps reset secrets in email only. Errors are returned to the queue consumer.
func (s *NotificationService) Dispatch(ctx context.Context, event model.NotificationEvent) error {
	if event.Type == "PASSWORD_RESET" {
		for _, strategy := range s.strategies {
			if strategy.Channel() == model.ChannelEmail {
				n := &model.Notification{UserID: event.UserID, Channel: model.ChannelEmail, Title: "Đặt lại mật khẩu UniHub", Content: "<p>Liên kết đặt lại mật khẩu có hiệu lực 30 phút:</p><a href=\"" + html.EscapeString(event.Metadata["reset_url"]) + "\">Đặt lại mật khẩu</a>"}
				return strategy.Send(ctx, n)
			}
		}
		return fmt.Errorf("email strategy unavailable")
	}
	// Ignore legacy reset messages containing plaintext passwords.
	if event.Type == "FORGOT_PASSWORD" {
		return nil
	}
	var registrationID *string
	if event.RegistrationID != "" {
		registrationID = &event.RegistrationID
	}
	title := "Đăng ký thành công: " + event.WorkshopTitle
	web := &model.Notification{UserID: event.UserID, RegistrationID: registrationID, Channel: model.ChannelWeb, Title: title, Content: "Vé workshop của bạn đã sẵn sàng.", Status: model.NotifSent, EventID: event.EventID, Link: "/"}
	if event.Metadata["email_only"] != "true" {
		if err := s.repo.Create(ctx, web); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	for _, strategy := range s.strategies {
		if strategy.Channel() == model.ChannelWeb {
			continue
		}
		n := *web
		n.Channel = strategy.Channel()
		n.Status = model.NotifPending
		n.Content = "<p>Bạn đã đăng ký workshop <strong>" + html.EscapeString(event.WorkshopTitle) + "</strong>.</p><p>Đăng nhập UniHub và mở mục thông báo để lấy vé QR.</p>"
		if err := s.repo.PrepareEmail(ctx, &n); err != nil {
			return err
		}
		if n.Status == model.NotifSent {
			continue
		}
		if err := strategy.Send(ctx, &n); err != nil {
			return err
		}
		if err := s.repo.UpdateStatus(ctx, n.ID, model.NotifSent, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *NotificationService) GetUserNotifications(ctx context.Context, userID string) ([]model.Notification, error) {
	return s.repo.FindByUser(ctx, userID)
}

// ==========================================
// EmailStrategy
// ==========================================

type EmailStrategy struct {
	host     string
	port     string
	from     string
	user     string
	password string
	userRepo *repository.UserRepo
}

func NewEmailStrategy(host, port, from, user, password string, userRepo *repository.UserRepo) *EmailStrategy {
	return &EmailStrategy{host: host, port: port, from: from, user: user, password: password, userRepo: userRepo}
}

func (e *EmailStrategy) Channel() model.NotificationChannel {
	return model.ChannelEmail
}

func (e *EmailStrategy) Send(ctx context.Context, notif *model.Notification) error {
	user, err := e.userRepo.FindByID(ctx, notif.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.Email == nil || *user.Email == "" {
		return fmt.Errorf("user %s has no email address", notif.UserID)
	}

	emailAddr := *user.Email
	// Format email with HTML headers
	subject := "Subject: " + strings.NewReplacer("\r", " ", "\n", " ").Replace(notif.Title) + "\r\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg := []byte(subject + mime + notif.Content)

	addr := fmt.Sprintf("%s:%s", e.host, e.port)

	// Setup authentication if credentials provided
	var auth smtp.Auth
	if e.user != "" && e.password != "" {
		auth = smtp.PlainAuth("", e.user, e.password, e.host)
	}

	return smtp.SendMail(addr, auth, e.from, []string{emailAddr}, msg)
}

// ==========================================
// WebNotificationStrategy
// ==========================================

type WebNotificationStrategy struct{}

func NewWebNotificationStrategy() *WebNotificationStrategy {
	return &WebNotificationStrategy{}
}

func (w *WebNotificationStrategy) Channel() model.NotificationChannel {
	return model.ChannelWeb
}

func (w *WebNotificationStrategy) Send(ctx context.Context, notif *model.Notification) error {
	// Web notifications are stored in DB and fetched by frontend polling/sockets
	log.Printf("[WEB_NOTIF] Notification queued in DB for user %s: %s", notif.UserID, notif.Title)
	return nil
}
