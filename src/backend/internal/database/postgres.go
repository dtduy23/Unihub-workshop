package database

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"unihub-workshop/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
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

	if cfg.DBMaxConns < 1 || cfg.DBMinConns < 0 || cfg.DBMinConns > cfg.DBMaxConns {
		log.Fatal("DB pool requires DB_MAX_CONNS >= 1 and 0 <= DB_MIN_CONNS <= DB_MAX_CONNS")
	}
	poolCfg.MaxConns = int32(cfg.DBMaxConns)
	poolCfg.MinConns = int32(cfg.DBMinConns)
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
