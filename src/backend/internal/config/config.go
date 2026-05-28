package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// App Mode: "api" | "worker" | "all" (default cho local dev)
	AppMode string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// RabbitMQ
	RabbitMQURL string

	// JWT & Crypto
	AuthSecret    string
	RSAPrivateKey string

	// Payment
	PaymentGatewayURL    string
	PaymentWebhookSecret string

	// AI
	AIApiURL    string
	AIApiKey    string
	GeminiModel string
	AITemperature float64
	AIMaxTokens   int

	// SMTP
	SMTPHost string
	SMTPPort string
	SMTPFrom string
	SMTPUser string
	SMTPPass string

	// Server
	ServerPort  string
	CORSOrigins string

	// Batch Import
	CSVImportDir  string
	CSVArchiveDir string

	// Rate Limiting
	RateLimitCapacity   int
	RateLimitRefillRate int
	RateLimitTTL        int
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppMode:    getEnv("APP_MODE", "all"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "unihub"),
		DBPassword: getEnv("DB_PASSWORD", "unihub_secret"),
		DBName:     getEnv("DB_NAME", "unihub_workshop"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),

		AuthSecret:    getEnv("AUTH_SECRET", "default-secret"),
		RSAPrivateKey: getEnv("RSA_PRIVATE_KEY", ""),

		PaymentGatewayURL:    getEnv("PAYMENT_GATEWAY_URL", "http://localhost:8080/mock/payment"),
		PaymentWebhookSecret: getEnv("PAYMENT_WEBHOOK_SECRET", "webhook-secret-key"),

		AIApiURL:    getEnv("AI_API_URL", "https://api.openai.com/v1/chat/completions"),
		AIApiKey:    getEnv("GEMINI_API_KEY", getEnv("AI_API_KEY", "")),
		GeminiModel: getEnv("GEMINI_MODEL", "gemini-1.5-flash-lite-preview"),
		AITemperature: getEnvFloat("AI_TEMPERATURE", 0.4),
		AIMaxTokens:   getEnvInt("AI_MAX_TOKENS", 800),

		SMTPHost: getEnv("SMTP_HOST", "localhost"),
		SMTPPort: getEnv("SMTP_PORT", "1025"),
		SMTPFrom: getEnv("SMTP_FROM", "noreply@unihub.edu.vn"),
		SMTPUser: getEnv("SMTP_USER", ""),
		SMTPPass: getEnv("SMTP_PASS", ""),

		ServerPort:  getEnv("SERVER_PORT", "8080"),
		CORSOrigins: getEnv("CORS_ORIGINS", "http://localhost:3000"),

		CSVImportDir:  getEnv("CSV_IMPORT_DIR", "./data/imports"),
		CSVArchiveDir: getEnv("CSV_ARCHIVE_DIR", "./data/archive"),

		RateLimitCapacity:   getEnvInt("RATE_LIMIT_CAPACITY", 50),
		RateLimitRefillRate: getEnvInt("RATE_LIMIT_REFILL_RATE", 5),
		RateLimitTTL:        getEnvInt("RATE_LIMIT_TTL", 300),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if val, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return fallback
}
