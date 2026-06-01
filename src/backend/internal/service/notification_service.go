package service

import (
	"context"
	"fmt"
	"log"
	"net/smtp"

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

// Dispatch sends notification through all configured channels
func (s *NotificationService) Dispatch(ctx context.Context, event model.NotificationEvent) {
	// 1. Create the primary DB record (for Web/In-app history)
	webNotif := &model.Notification{
		UserID:         event.UserID,
		RegistrationID: &event.RegistrationID,
		Channel:        model.ChannelWeb,
		Title:          buildTitle(event.Type, event.WorkshopTitle),
		Content:        buildContent(event),
		Status:         model.NotifSent, // Web notifications are considered 'sent' to DB immediately
		EventID:        event.EventID,
	}

	// Idempotent insert for Web notification
	if err := s.repo.Create(ctx, webNotif); err != nil {
		log.Printf("[NOTIFICATION] Web notification already exists for event %s, skipping", event.EventID)
		return
	}

	// 2. Process other external strategies (like Email) without creating additional DB rows
	for _, strategy := range strategies(s.strategies) {
		if strategy.Channel() == model.ChannelWeb {
			continue // Already created DB row above
		}

		// For other channels (Email), we just call Send
		// We use a temporary object for the strategy to process
		tempNotif := *webNotif
		tempNotif.Channel = strategy.Channel()

		go func(strat NotificationStrategy, n model.Notification) {
			if err := strat.Send(ctx, &n); err != nil {
				log.Printf("[NOTIFICATION] External dispatch failed via %s: %v", strat.Channel(), err)
			}
		}(strategy, tempNotif)
	}
}

func (s *NotificationService) GetUserNotifications(ctx context.Context, userID string) ([]model.Notification, error) {
	return s.repo.FindByUser(ctx, userID)
}

func strategies(s []NotificationStrategy) []NotificationStrategy { return s }

func buildTitle(eventType, workshopTitle string) string {
	switch eventType {
	case "REGISTRATION_SUCCESS":
		return fmt.Sprintf("🎉 Xác nhận đăng ký: %s", workshopTitle)
	case "PAYMENT_SUCCESS":
		return fmt.Sprintf("💳 Thanh toán thành công: %s", workshopTitle)
	case "FORGOT_PASSWORD":
		return "🔑 Cấp lại mật khẩu mới - UniHub"
	default:
		return fmt.Sprintf("🔔 Thông báo mới: %s", workshopTitle)
	}
}

func buildContent(event model.NotificationEvent) string {
	if event.Type == "FORGOT_PASSWORD" {
		newPassword := ""
		if event.Metadata != nil {
			newPassword = event.Metadata["new_password"]
		}
		
		return fmt.Sprintf(`
			<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #334155; line-height: 1.6;">
				<div style="background: #1e40af; padding: 30px; text-align: center; border-radius: 12px 12px 0 0;">
					<h1 style="color: white; margin: 0; font-size: 24px;">Yêu cầu cấp lại mật khẩu</h1>
				</div>
				<div style="padding: 30px; border: 1px solid #e2e8f0; border-top: none; border-radius: 0 0 12px 12px; background: white;">
					<h2 style="color: #0f172a; margin-top: 0;">Xin chào,</h2>
					<p>Hệ thống vừa nhận được yêu cầu cấp lại mật khẩu cho tài khoản của bạn.</p>
					
					<div style="background: #f0f7ff; padding: 20px; border-radius: 8px; border-left: 4px solid #1e40af; margin: 20px 0;">
						<p style="margin: 0; font-weight: bold; color: #1e40af;">Mật khẩu mới của bạn là:</p>
						<p style="margin: 10px 0 0 0; font-size: 24px; letter-spacing: 2px; font-family: monospace; background: #e2e8f0; padding: 10px; text-align: center; border-radius: 4px;">%s</p>
					</div>

					<p style="color: #dc2626; font-size: 14px; font-weight: bold;">Lưu ý: Mật khẩu cũ của bạn đã bị vô hiệu hóa.</p>
					<p>Vui lòng đăng nhập lại bằng mật khẩu mới này và chủ động đổi sang mật khẩu dễ nhớ hơn trong phần "Tài khoản của tôi".</p>
					<div style="text-align: center; margin-top: 30px;">
						<a href="http://localhost:3000/login" style="background: #1e40af; color: white; padding: 12px 30px; text-decoration: none; border-radius: 8px; font-weight: bold; display: inline-block;">Đăng nhập ngay</a>
					</div>
				</div>
				<p style="font-size: 12px; color: #94a3b8; text-align: center; margin-top: 20px;">Đây là email tự động từ UniHub Workshop Management System.</p>
			</div>
		`, newPassword)
	}

	if event.Type == "REGISTRATION_SUCCESS" || event.Type == "PAYMENT_SUCCESS" {
		isPaid := event.Type == "PAYMENT_SUCCESS"
		paymentLine := ""
		if isPaid {
			paymentLine = `<p style="color: #059669; font-weight: bold; margin-bottom: 20px;">✓ Hệ thống đã xác nhận bạn hoàn tất thanh toán thành công.</p>`
		}

		return fmt.Sprintf(`
			<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; color: #334155; line-height: 1.6;">
				<div style="background: #1e40af; padding: 30px; text-align: center; border-radius: 12px 12px 0 0;">
					<h1 style="color: white; margin: 0; font-size: 24px;">Xác nhận đăng ký thành công</h1>
				</div>
				<div style="padding: 30px; border: 1px solid #e2e8f0; border-top: none; border-radius: 0 0 12px 12px; background: white;">
					<h2 style="color: #0f172a; margin-top: 0;">Chúc mừng bạn!</h2>
					<p>Bạn đã đăng ký thành công workshop <strong>"%s"</strong>.</p>
					
					%s

					<div style="background: #f0f7ff; padding: 20px; border-radius: 8px; border-left: 4px solid #1e40af; margin: 20px 0;">
						<p style="margin: 0; font-weight: bold; color: #1e40af;">Hướng dẫn lấy mã QR:</p>
						<p style="margin: 10px 0 0 0; font-size: 14px;">Vì lý do bảo mật, mã QR check-in không được gửi qua email. Vui lòng truy cập vào <strong>Ứng dụng UniHub</strong> hoặc trang web để lấy mã vé của bạn.</p>
					</div>

					<p>Hẹn gặp bạn tại buổi Workshop!</p>
					<div style="text-align: center; margin-top: 30px;">
						<a href="http://localhost:3000" style="background: #1e40af; color: white; padding: 12px 30px; text-decoration: none; border-radius: 8px; font-weight: bold; display: inline-block;">Đến trang cá nhân</a>
					</div>
				</div>
				<p style="font-size: 12px; color: #94a3b8; text-align: center; margin-top: 20px;">Đây là email tự động từ UniHub Workshop Management System.</p>
			</div>
		`, event.WorkshopTitle, paymentLine)
	}

	return fmt.Sprintf("<p>Bạn có cập nhật mới về workshop <strong>\"%s\"</strong>.</p>", event.WorkshopTitle)
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
	subject := "Subject: " + notif.Title + "\r\n"
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
