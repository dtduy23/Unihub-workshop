package database

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// RunMigrations executes all pending SQL migrations embedded in the binary.
// It uses a `schema_migrations` table to track which migrations have already been applied.
// Each migration runs in a transaction — if it fails, it rolls back cleanly.
//
// Migration 002_seed_data.sql only runs when RUN_SEED=true.
func RunMigrations(pool *pgxpool.Pool) error {
	ctx := context.Background()

	// Tạo bảng tracking migration nếu chưa tồn tại
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Đọc danh sách file migration từ binary
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sắp xếp theo tên file (001_, 002_, ...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version := entry.Name()

		// Kiểm tra seed data: chỉ chạy khi RUN_SEED=true
		if strings.Contains(version, "seed") {
			runSeed := os.Getenv("RUN_SEED")
			if runSeed != "true" {
				log.Printf("[MIGRATION] Skipping %s (RUN_SEED != true)", version)
				continue
			}
		}

		// Kiểm tra migration đã chạy chưa
		var exists bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			version,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", version, err)
		}
		if exists {
			log.Printf("[MIGRATION] Already applied: %s", version)
			continue
		}

		// Đọc nội dung SQL
		content, err := migrationFiles.ReadFile("migrations/" + version)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", version, err)
		}

		// Chạy trong transaction
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", version, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		// Ghi nhận migration đã chạy
		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)",
			version,
		); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}

		log.Printf("[MIGRATION] Applied: %s", version)
	}

	log.Println("[MIGRATION] All migrations up to date")
	return nil
}
