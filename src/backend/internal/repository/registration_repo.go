package repository

import (
	"context"
	"fmt"
	"time"

	"unihub-workshop/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RegistrationRepo struct {
	pool *pgxpool.Pool
}

func NewRegistrationRepo(pool *pgxpool.Pool) *RegistrationRepo {
	return &RegistrationRepo{pool: pool}
}

func (r *RegistrationRepo) Create(ctx context.Context, tx pgx.Tx, reg *model.Registration) error {
	return tx.QueryRow(ctx,
		`INSERT INTO registrations (user_id, workshop_id, status, ticket_signature)
		 VALUES ($1, $2, $3, $4) 
		 ON CONFLICT (user_id, workshop_id) 
		 DO UPDATE SET status = EXCLUDED.status, ticket_signature = EXCLUDED.ticket_signature, created_at = NOW(),updated_at=NOW(),is_checked_in=false,checked_in_at=NULL
		 WHERE registrations.status NOT IN ('SUCCESS')
		 RETURNING id, created_at`,
		reg.UserID, reg.WorkshopID, reg.Status, reg.TicketSignature,
	).Scan(&reg.ID, &reg.CreatedAt)
}

func (r *RegistrationRepo) CreateDirect(ctx context.Context, reg *model.Registration) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO registrations (user_id, workshop_id, status, ticket_signature)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		reg.UserID, reg.WorkshopID, reg.Status, reg.TicketSignature,
	).Scan(&reg.ID, &reg.CreatedAt)
}

func (r *RegistrationRepo) FindByID(ctx context.Context, id string) (*model.Registration, error) {
	var reg model.Registration
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, workshop_id, status, ticket_signature, is_checked_in, created_at, updated_at, checked_in_at
		 FROM registrations WHERE id = $1`, id,
	).Scan(&reg.ID, &reg.UserID, &reg.WorkshopID, &reg.Status, &reg.TicketSignature,
		&reg.IsCheckedIn, &reg.CreatedAt, &reg.UpdatedAt, &reg.CheckedInAt)
	if err != nil {
		return nil, fmt.Errorf("registration not found: %w", err)
	}
	return &reg, nil
}

func (r *RegistrationRepo) FindByUserAndWorkshop(ctx context.Context, userID, workshopID string) (*model.Registration, error) {
	var reg model.Registration
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, workshop_id, status, ticket_signature, is_checked_in, created_at, updated_at, checked_in_at
		 FROM registrations WHERE user_id = $1 AND workshop_id = $2`, userID, workshopID,
	).Scan(&reg.ID, &reg.UserID, &reg.WorkshopID, &reg.Status, &reg.TicketSignature,
		&reg.IsCheckedIn, &reg.CreatedAt, &reg.UpdatedAt, &reg.CheckedInAt)
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *RegistrationRepo) FindByUser(ctx context.Context, userID string) ([]model.Registration, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, workshop_id, status, ticket_signature, is_checked_in, created_at, updated_at, checked_in_at
		 FROM registrations WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regs []model.Registration
	for rows.Next() {
		var reg model.Registration
		if err := rows.Scan(&reg.ID, &reg.UserID, &reg.WorkshopID, &reg.Status, &reg.TicketSignature,
			&reg.IsCheckedIn, &reg.CreatedAt, &reg.UpdatedAt, &reg.CheckedInAt); err != nil {
			return nil, err
		}
		regs = append(regs, reg)
	}
	return regs, nil
}

func (r *RegistrationRepo) FindByUserWithWorkshop(ctx context.Context, userID string) ([]model.RegistrationWithWorkshop, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.user_id, r.workshop_id, r.status, r.ticket_signature, r.is_checked_in, r.created_at,
		        w.title, w.room, w.start_time, w.end_time
		 FROM registrations r
		 JOIN workshops w ON r.workshop_id = w.id
		 WHERE r.user_id = $1
		 ORDER BY r.created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.RegistrationWithWorkshop
	for rows.Next() {
		var res model.RegistrationWithWorkshop
		if err := rows.Scan(&res.ID, &res.UserID, &res.WorkshopID, &res.Status, &res.TicketSignature,
			&res.IsCheckedIn, &res.CreatedAt,
			&res.WorkshopTitle, &res.WorkshopRoom, &res.StartTime, &res.EndTime); err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

func (r *RegistrationRepo) UpdateStatus(ctx context.Context, id string, status model.RegistrationStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE registrations SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (r *RegistrationRepo) UpdateStatusAndQR(ctx context.Context, id string, status model.RegistrationStatus, qrCode string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE registrations SET status = $1, ticket_signature = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`, status, qrCode, id)
	return err
}

func (r *RegistrationRepo) CheckIn(ctx context.Context, id string) error {
	return r.CheckInWithTime(ctx, id, time.Now().Unix())
}

func (r *RegistrationRepo) CheckInWithTime(ctx context.Context, id string, scannedAt int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE registrations r SET is_checked_in=TRUE, checked_in_at=LEAST(COALESCE(checked_in_at,to_timestamp($1)),to_timestamp($1)),updated_at=now() WHERE r.id=$2 AND r.status='SUCCESS' AND EXISTS(SELECT 1 FROM workshops w WHERE w.id=r.workshop_id AND w.status IN ('PUBLISHED','CLOSED'))`,
		scannedAt, id)
	if err == nil && tag.RowsAffected() == 0 {
		return fmt.Errorf("vé không còn hiệu lực")
	}
	return err
}

func (r *RegistrationRepo) FindByStudentAndWorkshop(ctx context.Context, studentID, workshopID string) (*model.Registration, error) {
	var reg model.Registration
	err := r.pool.QueryRow(ctx,
		`SELECT r.id, r.user_id, r.workshop_id, r.status, r.ticket_signature, r.is_checked_in, r.created_at, r.updated_at, r.checked_in_at
		 FROM registrations r
		 JOIN users u ON r.user_id = u.id
		 WHERE u.user_id = $1 AND r.workshop_id = $2 AND r.status = 'SUCCESS'`,
		studentID, workshopID,
	).Scan(&reg.ID, &reg.UserID, &reg.WorkshopID, &reg.Status, &reg.TicketSignature,
		&reg.IsCheckedIn, &reg.CreatedAt, &reg.UpdatedAt, &reg.CheckedInAt)
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *RegistrationRepo) FindByWorkshopWithUser(ctx context.Context, workshopID string) ([]model.RegistrationWithUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.user_id, r.workshop_id, r.status, r.ticket_signature, r.is_checked_in, r.created_at,
		        u.user_id AS student_id, u.full_name, u.email
		 FROM registrations r
		 JOIN users u ON r.user_id = u.id
		 WHERE r.workshop_id = $1
		 ORDER BY r.created_at DESC`, workshopID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.RegistrationWithUser
	for rows.Next() {
		var res model.RegistrationWithUser
		if err := rows.Scan(&res.ID, &res.UserID, &res.WorkshopID, &res.Status, &res.TicketSignature,
			&res.IsCheckedIn, &res.CreatedAt,
			&res.StudentID, &res.FullName, &res.Email); err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

func (r *RegistrationRepo) GetPool() *pgxpool.Pool {
	return r.pool
}
func (r *RegistrationRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM registrations WHERE id = $1`, id)
	return err
}
