package database

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"unihub-workshop/internal/config"
)

func NewPostgresPool(cfg *config.Config) *pgxpool.Pool {
	// Sử dụng net/url để encode password an toàn (tránh lỗi ký tự đặc biệt như @)
	userInfo := url.UserPassword(cfg.DBUser, cfg.DBPassword)
	host := fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort)
	
	u := url.URL{
		Scheme:   "postgres",
		User:     userInfo,
		Host:     host,
		Path:     cfg.DBName,
		RawQuery: fmt.Sprintf("sslmode=%s&timezone=Asia/Ho_Chi_Minh", cfg.DBSSLMode),
	}
	
	dsn := u.String()

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Unable to parse database config: %v", err)
	}

	poolCfg.MaxConns = 20
	poolCfg.MinConns = 5
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Println("[DB] PostgreSQL connection pool established")
	return pool
}
