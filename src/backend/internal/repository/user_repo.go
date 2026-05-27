package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"unihub-workshop/internal/model"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) FindByStudentID(ctx context.Context, identifier string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, password_hash, full_name, email, phone, role, created_at, updated_at
		 FROM users WHERE user_id = $1 OR email = $1`, identifier,
	).Scan(&u.ID, &u.StudentID, &u.PasswordHash, &u.FullName, &u.Email, &u.Phone, &u.Role, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, password_hash, full_name, email, phone, role, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.StudentID, &u.PasswordHash, &u.FullName, &u.Email, &u.Phone, &u.Role, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) UpsertFromCSV(ctx context.Context, studentID, passwordHash, fullName, email, phone, role string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (user_id, password_hash, full_name, email, phone, role)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id) DO UPDATE SET
		   password_hash = EXCLUDED.password_hash,
		   full_name = EXCLUDED.full_name,
		   email = EXCLUDED.email,
		   phone = EXCLUDED.phone,
		   role = EXCLUDED.role,
		   updated_at = CURRENT_TIMESTAMP`,
		studentID, passwordHash, fullName, email, phone, role,
	)
	return err
}

func (r *UserRepo) GetStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)
	rows, err := r.pool.Query(ctx, `SELECT role, COUNT(*) FROM users GROUP BY role`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var role string
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			return nil, err
		}
		stats[role] = count
	}
	return stats, nil
}
