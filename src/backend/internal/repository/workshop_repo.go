package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
	"unihub-workshop/internal/model"
)

type WorkshopRepo struct{ pool *pgxpool.Pool }

func NewWorkshopRepo(pool *pgxpool.Pool) *WorkshopRepo { return &WorkshopRepo{pool: pool} }

const workshopSelect = `SELECT to_jsonb(w)||jsonb_build_object('company_name',COALESCE(c.name,'')) FROM workshops w LEFT JOIN companies c ON c.id=w.company_id `

func scanWorkshop(row pgx.Row) (*model.Workshop, error) {
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, err
	}
	var w model.Workshop
	err := json.Unmarshal(data, &w)
	return &w, err
}
func (r *WorkshopRepo) FindAll(ctx context.Context, title string) ([]model.Workshop, error) {
	return r.List(ctx, title, false, "")
}
func (r *WorkshopRepo) List(ctx context.Context, title string, admin bool, company string) ([]model.Workshop, error) {
	query := workshopSelect + `WHERE w.title ILIKE $1`
	args := []any{"%" + title + "%"}
	if company != "" {
		args = append(args, company)
		query += " AND w.company_id=$2"
	} else if !admin {
		query += ` AND w.status IN ('PUBLISHED','CLOSED','CANCELLED') AND (w.company_id IS NULL OR c.status='APPROVED')`
	}
	rows, err := r.pool.Query(ctx, query+" ORDER BY w.start_time,w.id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.Workshop{}
	for rows.Next() {
		w, err := scanWorkshop(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *w)
	}
	return result, rows.Err()
}
func (r *WorkshopRepo) FindByID(ctx context.Context, id string) (*model.Workshop, error) {
	return scanWorkshop(r.pool.QueryRow(ctx, workshopSelect+"WHERE w.id=$1", id))
}

func (r *WorkshopRepo) Assigned(ctx context.Context, user string, admin bool) ([]model.Workshop, error) {
	query := workshopSelect + `WHERE w.status IN ('PUBLISHED','CLOSED') AND (w.company_id IS NULL OR c.status='APPROVED')`
	args := []any{}
	if !admin {
		query += ` AND EXISTS(SELECT 1 FROM workshop_staff ws WHERE ws.workshop_id=w.id AND ws.user_id=$1)`
		args = append(args, user)
	}
	rows, err := r.pool.Query(ctx, query+" ORDER BY w.start_time,w.id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.Workshop{}
	for rows.Next() {
		w, e := scanWorkshop(rows)
		if e != nil {
			return nil, e
		}
		result = append(result, *w)
	}
	return result, rows.Err()
}
func (r *WorkshopRepo) Create(ctx context.Context, w *model.Workshop) error {
	return r.pool.QueryRow(ctx, `INSERT INTO workshops(title,description,speaker,room,start_time,end_time,capacity,available_seats,price,status,summary,room_layout_url,registration_start_time,registration_end_time,company_id,created_by,cover_url,audience,benefits,preparation,agenda,format)
 VALUES($1,$2,$3,$4,$5,$6,$7,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21) RETURNING id,created_at`, w.Title, w.Description, w.Speaker, w.Room, w.StartTime, w.EndTime, w.Capacity, w.Price, w.Status, w.Summary, w.RoomLayoutURL, w.RegistrationStartTime, w.RegistrationEndTime, w.CompanyID, w.CreatedBy, w.CoverURL, w.Audience, w.Benefits, w.Preparation, w.Agenda, w.Format).Scan(&w.ID, &w.CreatedAt)
}
func (r *WorkshopRepo) Update(ctx context.Context, id string, req *model.UpdateWorkshopRequest) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var capacity, available int
	if err = tx.QueryRow(ctx, "SELECT capacity,available_seats FROM workshops WHERE id=$1 FOR UPDATE", id).Scan(&capacity, &available); err != nil {
		return err
	}
	clauses := []string{}
	args := []any{}
	add := func(name string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s=$%d", name, len(args)))
	}
	if req.Title != nil {
		add("title", *req.Title)
	}
	if req.Description != nil {
		add("description", *req.Description)
	}
	if req.Speaker != nil {
		add("speaker", *req.Speaker)
	}
	if req.Room != nil {
		add("room", *req.Room)
	}
	for _, field := range []struct {
		name  string
		value *string
	}{
		{"start_time", req.StartTime}, {"end_time", req.EndTime},
		{"registration_start_time", req.RegistrationStartTime}, {"registration_end_time", req.RegistrationEndTime},
	} {
		if field.value != nil {
			value, err := time.Parse(time.RFC3339, *field.value)
			if err != nil {
				return err
			}
			add(field.name, value)
		}
	}
	if req.Capacity != nil {
		if *req.Capacity < capacity-available {
			return fmt.Errorf("sức chứa nhỏ hơn số ghế đã giữ")
		}
		add("capacity", *req.Capacity)
		add("available_seats", available+*req.Capacity-capacity)
	}
	if req.Price != nil {
		add("price", *req.Price)
	}
	if req.Status != nil {
		add("status", *req.Status)
	}
	if req.Summary != nil {
		add("summary", *req.Summary)
	}
	if req.RoomLayoutURL != nil {
		add("room_layout_url", *req.RoomLayoutURL)
	}
	if req.CoverURL != nil {
		add("cover_url", *req.CoverURL)
	}
	if req.Audience != nil {
		add("audience", *req.Audience)
	}
	if req.Benefits != nil {
		add("benefits", *req.Benefits)
	}
	if req.Preparation != nil {
		add("preparation", *req.Preparation)
	}
	if req.Agenda != nil {
		add("agenda", *req.Agenda)
	}
	if req.Format != nil {
		add("format", *req.Format)
	}
	if len(args) == 0 {
		return fmt.Errorf("no fields to update")
	}
	clauses = append(clauses, "updated_at=now()")
	args = append(args, id)
	if _, err = tx.Exec(ctx, "UPDATE workshops SET "+strings.Join(clauses, ",")+fmt.Sprintf(" WHERE id=$%d", len(args)), args...); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *WorkshopRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "UPDATE workshops SET status='DELETED',updated_at=now() WHERE id=$1", id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *WorkshopRepo) DecrementSeatWithLock(ctx context.Context, tx pgx.Tx, id string) (int, error) {
	var seats int
	err := tx.QueryRow(ctx, `UPDATE workshops SET available_seats=available_seats-1 WHERE id=$1 AND available_seats>0 AND status='PUBLISHED' AND registration_start_time<=now() AND registration_end_time>=now() RETURNING available_seats`, id).Scan(&seats)
	return seats, err
}
func (r *WorkshopRepo) IncrementSeat(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE workshops SET available_seats=LEAST(available_seats+1,capacity) WHERE id=$1", id)
	return err
}
func (r *WorkshopRepo) UpdateSummary(ctx context.Context, id, summary string) error {
	tag, err := r.pool.Exec(ctx, "UPDATE workshops SET summary=$1 WHERE id=$2", summary, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *WorkshopRepo) GetPool() *pgxpool.Pool { return r.pool }
