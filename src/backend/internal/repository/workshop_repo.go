package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"unihub-workshop/internal/model"
)

type WorkshopRepo struct {
	pool *pgxpool.Pool
}

func NewWorkshopRepo(pool *pgxpool.Pool) *WorkshopRepo {
	return &WorkshopRepo{pool: pool}
}

func (r *WorkshopRepo) FindAll(ctx context.Context, title string) ([]model.Workshop, error) {
	query := `SELECT id, title, description, speaker, room, start_time, end_time,
		        capacity, available_seats, price, summary, status, room_layout_url, created_at,
		        registration_start_time, registration_end_time
		 FROM workshops WHERE 1=1`
	
	args := []interface{}{}
	if title != "" {
		query += " AND title ILIKE $1"
		args = append(args, "%"+title+"%")
	}
	
	query += " ORDER BY start_time ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workshops []model.Workshop
	for rows.Next() {
		var w model.Workshop
		if err := rows.Scan(&w.ID, &w.Title, &w.Description, &w.Speaker, &w.Room,
			&w.StartTime, &w.EndTime, &w.Capacity, &w.AvailableSeats, &w.Price,
			&w.Summary, &w.Status, &w.RoomLayoutURL, &w.CreatedAt,
			&w.RegistrationStartTime, &w.RegistrationEndTime); err != nil {
			return nil, err
		}
		workshops = append(workshops, w)
	}
	return workshops, nil
}

func (r *WorkshopRepo) FindByID(ctx context.Context, id string) (*model.Workshop, error) {
	var w model.Workshop
	err := r.pool.QueryRow(ctx,
		`SELECT id, title, description, speaker, room, start_time, end_time,
		        capacity, available_seats, price, summary, status, room_layout_url, created_at,
		        registration_start_time, registration_end_time
		 FROM workshops WHERE id = $1`, id,
	).Scan(&w.ID, &w.Title, &w.Description, &w.Speaker, &w.Room,
		&w.StartTime, &w.EndTime, &w.Capacity, &w.AvailableSeats, &w.Price,
		&w.Summary, &w.Status, &w.RoomLayoutURL, &w.CreatedAt,
		&w.RegistrationStartTime, &w.RegistrationEndTime)
	if err != nil {
		return nil, fmt.Errorf("workshop not found: %w", err)
	}
	return &w, nil
}

func (r *WorkshopRepo) Create(ctx context.Context, w *model.Workshop) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO workshops (title, speaker, room, start_time, end_time, capacity, available_seats, price, status, summary, room_layout_url, registration_start_time, registration_end_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $8, $9, $10, $11, $12) RETURNING id, created_at`,
		w.Title, w.Speaker, w.Room, w.StartTime, w.EndTime, w.Capacity, w.Price, w.Status, w.Summary, w.RoomLayoutURL, w.RegistrationStartTime, w.RegistrationEndTime,
	).Scan(&w.ID, &w.CreatedAt)
}

func (r *WorkshopRepo) Update(ctx context.Context, id string, req *model.UpdateWorkshopRequest) error {
	// Build dynamic update query
	query := "UPDATE workshops SET "
	args := []interface{}{}
	argIdx := 1
	setClauses := []string{}

	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.Speaker != nil {
		setClauses = append(setClauses, fmt.Sprintf("speaker = $%d", argIdx))
		args = append(args, *req.Speaker)
		argIdx++
	}
	if req.Room != nil {
		setClauses = append(setClauses, fmt.Sprintf("room = $%d", argIdx))
		args = append(args, *req.Room)
		argIdx++
	}
	if req.StartTime != nil {
		setClauses = append(setClauses, fmt.Sprintf("start_time = $%d", argIdx))
		args = append(args, *req.StartTime)
		argIdx++
	}
	if req.EndTime != nil {
		setClauses = append(setClauses, fmt.Sprintf("end_time = $%d", argIdx))
		args = append(args, *req.EndTime)
		argIdx++
	}
	if req.RegistrationStartTime != nil {
		setClauses = append(setClauses, fmt.Sprintf("registration_start_time = $%d", argIdx))
		args = append(args, *req.RegistrationStartTime)
		argIdx++
	}
	if req.RegistrationEndTime != nil {
		setClauses = append(setClauses, fmt.Sprintf("registration_end_time = $%d", argIdx))
		args = append(args, *req.RegistrationEndTime)
		argIdx++
	}
	if req.Capacity != nil {
		setClauses = append(setClauses, fmt.Sprintf("capacity = $%d", argIdx))
		args = append(args, *req.Capacity)
		argIdx++
		setClauses = append(setClauses, fmt.Sprintf("available_seats = $%d", argIdx))
		args = append(args, *req.Capacity)
		argIdx++
	}
	if req.Price != nil {
		setClauses = append(setClauses, fmt.Sprintf("price = $%d", argIdx))
		args = append(args, *req.Price)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.Summary != nil {
		setClauses = append(setClauses, fmt.Sprintf("summary = $%d", argIdx))
		args = append(args, *req.Summary)
		argIdx++
	}
	if req.RoomLayoutURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("room_layout_url = $%d", argIdx))
		args = append(args, *req.RoomLayoutURL)
		argIdx++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields to update")
	}

	for i, c := range setClauses {
		query += c
		if i < len(setClauses)-1 {
			query += ", "
		}
	}
	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *WorkshopRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE workshops SET status = 'DELETED' WHERE id = $1`, id)
	return err
}

// DecrementSeatWithLock uses Pessimistic Locking (SELECT FOR UPDATE) to safely decrement seats
func (r *WorkshopRepo) DecrementSeatWithLock(ctx context.Context, tx pgx.Tx, workshopID string) (int, error) {
	var availableSeats int
	err := tx.QueryRow(ctx,
		`SELECT available_seats FROM workshops WHERE id = $1 FOR UPDATE`,
		workshopID,
	).Scan(&availableSeats)
	if err != nil {
		return 0, fmt.Errorf("failed to lock workshop: %w", err)
	}

	if availableSeats <= 0 {
		return 0, fmt.Errorf("no available seats")
	}

	_, err = tx.Exec(ctx,
		`UPDATE workshops SET available_seats = available_seats - 1 WHERE id = $1`,
		workshopID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to decrement seats: %w", err)
	}

	return availableSeats - 1, nil
}

// IncrementSeat restores a seat (used when payment times out)
func (r *WorkshopRepo) IncrementSeat(ctx context.Context, workshopID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workshops SET available_seats = LEAST(available_seats + 1, capacity) WHERE id = $1`,
		workshopID,
	)
	return err
}

func (r *WorkshopRepo) UpdateSummary(ctx context.Context, workshopID, summary string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workshops SET summary = $1 WHERE id = $2`,
		summary, workshopID,
	)
	return err
}

// GetPool exposes the pool for transaction management
func (r *WorkshopRepo) GetPool() *pgxpool.Pool {
	return r.pool
}
