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
		`SELECT id, user_id, password_hash, full_name, email, phone, role, created_at, updated_at, auth_version
		 FROM users WHERE user_id = $1 OR email = $1`, identifier,
	).Scan(&u.ID, &u.StudentID, &u.PasswordHash, &u.FullName, &u.Email, &u.Phone, &u.Role, &u.CreatedAt, &u.UpdatedAt, &u.AuthVersion)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, password_hash, full_name, email, phone, role, created_at, updated_at, auth_version
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.StudentID, &u.PasswordHash, &u.FullName, &u.Email, &u.Phone, &u.Role, &u.CreatedAt, &u.UpdatedAt, &u.AuthVersion)

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

func (r *UserRepo) UpdatePassword(ctx context.Context, userID, newPasswordHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, auth_version=auth_version+1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		newPasswordHash, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
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

func (r *UserRepo) StoreReset(ctx context.Context, userID, hash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "DELETE FROM password_resets WHERE user_id=$1 OR expires_at<now()", userID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO password_resets(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '30 minutes')", hash, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *UserRepo) ConsumeReset(ctx context.Context, hash, passwordHash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id string
	if err = tx.QueryRow(ctx, "DELETE FROM password_resets WHERE token_hash=$1 AND expires_at>now() RETURNING user_id", hash).Scan(&id); err != nil {
		return fmt.Errorf("liên kết không hợp lệ hoặc đã hết hạn")
	}
	if _, err = tx.Exec(ctx, "UPDATE users SET password_hash=$1,auth_version=auth_version+1,updated_at=now() WHERE id=$2", passwordHash, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM password_resets WHERE user_id=$1", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
