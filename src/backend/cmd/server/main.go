package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"unihub-workshop/internal/config"
	"unihub-workshop/internal/crypto"
	"unihub-workshop/internal/database"
	"unihub-workshop/internal/handler"
	"unihub-workshop/internal/logger"
	"unihub-workshop/internal/metrics"
	"unihub-workshop/internal/middleware"
	"unihub-workshop/internal/model"
	"unihub-workshop/internal/queue"
	"unihub-workshop/internal/ratelimiter"
	"unihub-workshop/internal/repository"
	"unihub-workshop/internal/service"
)

func main() {
	// Initialize structured JSON logging (must be first)
	logger.Init()

	// Set default timezone to Vietnam
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		log.Printf("[CONFIG] Warning: Failed to load Asia/Ho_Chi_Minh location: %v", err)
	} else {
		time.Local = loc
		log.Println("[CONFIG] System timezone set to Asia/Ho_Chi_Minh")
	}

	// Load config
	cfg := config.Load()

	// Initialize infrastructure
	pgPool := database.NewPostgresPool(cfg)
	defer pgPool.Close()

	// Auto-run database migrations
	if err := database.RunMigrations(pgPool); err != nil {
		log.Fatalf("[MIGRATION] Failed: %v", err)
	}

	// Start DB pool metrics collector (every 15s)
	metrics.StartDBCollector(pgPool, 15*time.Second)

	redisClient := database.NewRedisClient(cfg)
	defer redisClient.Close()

	publisher, err := queue.NewPublisher(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	consumer, err := queue.NewConsumer(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepo(pgPool)
	workshopRepo := repository.NewWorkshopRepo(pgPool)
	regRepo := repository.NewRegistrationRepo(pgPool)
	paymentRepo := repository.NewPaymentRepo(pgPool)
	notifRepo := repository.NewNotificationRepo(pgPool)
	importRepo := repository.NewImportRepo(pgPool)

	// Initialize waiting room (max 100 concurrent registrations, token valid for 5 min, queue valid for 1 hour)
	waitingRoom := ratelimiter.NewWaitingRoom(redisClient, 100, 300, 3600)

	// Initialize seat limiter (Double-check pattern)
	seatLimiter := ratelimiter.NewSeatLimiter(redisClient)

	// Initialize RSA Crypto Provider
	var rsaProvider *crypto.RSAProvider
	if cfg.RSAPrivateKey != "" {
		var err error
		rsaProvider, err = crypto.NewRSAProvider(cfg.RSAPrivateKey)
		if err != nil {
			log.Printf("[CRYPTO] Warning: Failed to initialize RSA provider: %v", err)
		} else {
			log.Println("[CRYPTO] RSA provider initialized successfully")
		}
	} else {
		log.Println("[CRYPTO] Warning: RSA_PRIVATE_KEY is empty, signing will be disabled")
	}

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.AuthSecret, publisher)
	workshopService := service.NewWorkshopService(workshopRepo)
	paymentService := service.NewPaymentService(paymentRepo, regRepo, workshopRepo, userRepo, rsaProvider, publisher, redisClient, seatLimiter, cfg.PaymentWebhookSecret, cfg.PaymentGatewayURL)
	regService := service.NewRegistrationService(regRepo, workshopRepo, userRepo, paymentService, rsaProvider, publisher, redisClient, waitingRoom, seatLimiter)
	checkinService := service.NewCheckinService(regRepo)

	// Notification strategies (Strategy + Observer Pattern)
	emailStrategy := service.NewEmailStrategy(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom, cfg.SMTPUser, cfg.SMTPPass, userRepo)
	webStrategy := service.NewWebNotificationStrategy()
	notifService := service.NewNotificationService(notifRepo, emailStrategy, webStrategy)

	batchService := service.NewBatchImportService(importRepo, userRepo, cfg.CSVImportDir, cfg.CSVArchiveDir)
	aiService := service.NewAISummaryService(workshopRepo, cfg.AIApiKey, cfg.GeminiModel, cfg.AITemperature, cfg.AIMaxTokens)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, rsaProvider)
	workshopHandler := handler.NewWorkshopHandler(workshopService)
	regHandler := handler.NewRegistrationHandler(regService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	checkinHandler := handler.NewCheckinHandler(checkinService)
	notifHandler := handler.NewNotificationHandler(notifService)
	adminHandler := handler.NewAdminHandler(batchService, aiService, userRepo, cfg.CSVImportDir)
	aiHandler := handler.NewAIHandler(aiService)

	// Initialize rate limiter
	redisBucket := ratelimiter.NewRedisTokenBucket(redisClient, cfg.RateLimitCapacity, cfg.RateLimitRefillRate, cfg.RateLimitTTL)
	localBucket := ratelimiter.NewLocalBucket(cfg.RateLimitCapacity, cfg.RateLimitRefillRate, 10000)
	rateLimitMW := middleware.NewRateLimitMiddleware(redisBucket, localBucket)

	// Build router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.StructuredLogger)
	r.Use(middleware.MetricsMiddleware)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.CORSMiddleware(cfg.CORSOrigins))

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	// Public routes
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Get("/api/v1/auth/public-key", authHandler.GetPublicKey)
	r.Post("/api/v1/auth/forgot-password", authHandler.ForgotPassword)

	// Payment webhook (public, signature-verified)
	r.Post("/api/v1/payment/webhook", paymentHandler.Webhook)

	// Mock Gateway Simulation (for testing)
	mockHandler := handler.NewMockHandler(redisClient)
	r.Post("/api/v1/mock/payment/toggle", mockHandler.ToggleGatewayStatus)
	r.Get("/api/v1/mock/payment/status", mockHandler.GetGatewayStatus)

	// Mock payment endpoint for testing
	r.Get("/mock/payment/checkout", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.APIResponse{
			Success: true,
			Message: "Mock payment page - use webhook to simulate payment result",
			Data: map[string]string{
				"transaction_id": r.URL.Query().Get("tx"),
				"instruction":    "POST to /api/v1/payment/webhook with transaction_id, status, and signature",
			},
		})
	})

	// Public workshop listing (no auth needed for browsing)
	r.Get("/api/v1/workshops", workshopHandler.List)
	r.Get("/api/v1/workshops/{id}", workshopHandler.GetByID)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.AuthSecret))
		r.Use(rateLimitMW.Handler)

		// User profile
		r.Get("/api/v1/auth/me", authHandler.GetMe)
		r.Post("/api/v1/auth/change-password", authHandler.ChangePassword)

		// Notifications (all authenticated users)
		r.Get("/api/v1/notifications", notifHandler.GetMyNotifications)

		// Student routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole(model.RoleStudent, model.RoleAdmin))

			r.Post("/api/v1/registrations", regHandler.Register)
			r.Post("/api/v1/registrations/{id}/cancel", regHandler.Cancel)
			r.Get("/api/v1/registrations/waiting-room/{workshopId}", regHandler.GetWaitingRoomStatus)
			r.Get("/api/v1/registrations/status/{correlationId}", regHandler.GetStatus)
			r.Get("/api/v1/registrations/my", regHandler.MyRegistrations)
			r.Post("/api/v1/payments/{registrationId}", paymentHandler.InitiatePayment)
			r.Get("/api/v1/payments/status/{transactionId}", paymentHandler.GetPaymentStatus)
		})

		// Staff routes (check-in)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole(model.RoleStaff, model.RoleAdmin))

			r.Post("/api/v1/checkin/live", checkinHandler.LiveCheckin)
			r.Post("/api/v1/checkin/sync", checkinHandler.BulkSync)
		})

		// Organizer (admin) routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole(model.RoleAdmin))

			r.Post("/api/v1/workshops", workshopHandler.Create)
			r.Put("/api/v1/workshops/{id}", workshopHandler.Update)
			r.Delete("/api/v1/workshops/{id}", workshopHandler.Delete)

			r.Post("/api/v1/admin/import/csv", adminHandler.UploadCSV)
			r.Get("/api/v1/admin/import/jobs", adminHandler.GetImportJobs)
			r.Get("/api/v1/admin/import/jobs/{id}/errors", adminHandler.GetImportErrors)
			r.Post("/api/v1/admin/import/jobs/{id}/run", adminHandler.RunJob)
			r.Post("/api/v1/admin/workshops/{workshopId}/summary", adminHandler.UploadPDF)
			r.Post("/api/v1/ai/summarize", aiHandler.SummarizePDF)
			r.Get("/api/v1/registrations/workshop/{workshopId}", regHandler.GetByWorkshopID)
			r.Get("/api/v1/registrations/workshop/{workshopId}/export", regHandler.ExportWorkshopCSV)
			r.Get("/api/v1/admin/stats", adminHandler.GetStats)
			r.Get("/api/v1/admin/payments/pending", paymentHandler.GetPendingPayments)
			r.Get("/api/v1/admin/payment/gateway-status", paymentHandler.GetGatewayStatus)
			r.Get("/api/v1/admin/payment/circuit-breaker", paymentHandler.GetCircuitBreakerStatus)
		})
	})

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mode := cfg.AppMode
	log.Printf("[SERVER] Running in mode: %s", mode)

	// Start background workers (chỉ khi mode = "worker" hoặc "all")
	if mode == "worker" || mode == "all" {
		go startRegistrationWorker(ctx, consumer, regService)
		go startNotificationWorker(ctx, cfg.RabbitMQURL, notifService)
		go startPaymentCleanupWorker(ctx, paymentService)
		go startBatchImportScheduler(ctx, batchService)
		log.Println("[SERVER] Background workers started")
	}

	// Start HTTP server (chỉ khi mode = "api" hoặc "all")
	if mode == "api" || mode == "all" {
		srv := &http.Server{
			Addr:         ":" + cfg.ServerPort,
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		// Graceful shutdown
		go func() {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			log.Println("[SERVER] Shutting down gracefully...")
			cancel()

			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()

			if err := srv.Shutdown(shutdownCtx); err != nil {
				log.Printf("[SERVER] Forced shutdown: %v", err)
			}
		}()

		log.Printf("[SERVER] Starting on port %s", cfg.ServerPort)
		log.Printf("[SERVER] API docs: http://localhost:%s/health", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[SERVER] Failed to start: %v", err)
		}
		log.Println("[SERVER] Stopped")
	} else {
		// Worker-only mode: block cho đến khi nhận tín hiệu tắt
		log.Println("[SERVER] Worker-only mode — waiting for shutdown signal...")
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("[SERVER] Worker shutting down...")
		cancel()
	}
}

// Background workers

func startRegistrationWorker(ctx context.Context, consumer *queue.Consumer, regService *service.RegistrationService) {
	msgs, err := consumer.Consume(queue.RegistrationQueue)
	if err != nil {
		log.Fatalf("[WORKER] Failed to start registration consumer: %v", err)
	}

	log.Println("[WORKER] Registration worker started")
	for {
		select {
		case <-ctx.Done():
			log.Println("[WORKER] Registration worker stopping")
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}

			var queueMsg model.QueueMessage
			if err := json.Unmarshal(msg.Body, &queueMsg); err != nil {
				log.Printf("[WORKER] Failed to unmarshal message: %v", err)
				msg.Nack(false, false) // Send to DLQ
				continue
			}

			if err := regService.ProcessRegistration(ctx, queueMsg); err != nil {
				log.Printf("[WORKER] Processing failed: %v", err)
				// Retry up to 3 times
				retryCount := 0
				if msg.Headers != nil {
					if rc, ok := msg.Headers["x-retry-count"].(int64); ok {
						retryCount = int(rc)
					}
				}
				if retryCount < 3 {
					msg.Nack(false, true) // Requeue
				} else {
					msg.Nack(false, false) // Send to DLQ
				}
				continue
			}

			msg.Ack(false)
		}
	}
}

func startNotificationWorker(ctx context.Context, rabbitURL string, notifService *service.NotificationService) {
	notifConsumer, err := queue.NewConsumer(rabbitURL)
	if err != nil {
		log.Printf("[WORKER] Failed to start notification consumer: %v", err)
		return
	}
	defer notifConsumer.Close()

	msgs, err := notifConsumer.Consume(queue.NotificationQueue)
	if err != nil {
		log.Printf("[WORKER] Failed to consume notifications: %v", err)
		return
	}

	log.Println("[WORKER] Notification worker started")
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}

			var event model.NotificationEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("[WORKER] Failed to unmarshal notification: %v", err)
				msg.Nack(false, false)
				continue
			}

			notifService.Dispatch(ctx, event)
			msg.Ack(false)
		}
	}
}

func startPaymentCleanupWorker(ctx context.Context, paymentService *service.PaymentService) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("[WORKER] Payment cleanup worker started (interval: 5min)")
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			paymentService.CleanupExpiredPayments(ctx)
		}
	}
}

func startBatchImportScheduler(ctx context.Context, batchService *service.BatchImportService) {
	log.Println("[WORKER] Batch import scheduler started (daily at 02:00)")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			now := time.Now()
			// Calculate next 02:00 AM
			next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			
			duration := next.Sub(now)
			log.Printf("[WORKER] Next batch import scheduled in %v (at %v)", duration.Round(time.Second), next.Format("15:04:05"))

			timer := time.NewTimer(duration)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				log.Println("[BATCH_IMPORT] Scheduled import triggered")
				batchService.ScanAndImport(ctx)
			}
		}
	}
}
