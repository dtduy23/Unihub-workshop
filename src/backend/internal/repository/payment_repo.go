package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"unihub-workshop/internal/model"
)

type PaymentRepo struct {
	pool *pgxpool.Pool
}

func NewPaymentRepo(pool *pgxpool.Pool) *PaymentRepo {
	return &PaymentRepo{pool: pool}
}

func (r *PaymentRepo) Create(ctx context.Context, p *model.Payment) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO payments (registration_id, transaction_id, amount, provider, status)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		p.RegistrationID, p.TransactionID, p.Amount, p.Provider, p.Status,
	).Scan(&p.ID, &p.CreatedAt)
}

func (r *PaymentRepo) FindByTransactionID(ctx context.Context, txID string) (*model.Payment, error) {
	var p model.Payment
	err := r.pool.QueryRow(ctx,
		`SELECT id, registration_id, transaction_id, amount, provider, status, created_at
		 FROM payments WHERE transaction_id = $1`, txID,
	).Scan(&p.ID, &p.RegistrationID, &p.TransactionID, &p.Amount, &p.Provider, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}
	return &p, nil
}

func (r *PaymentRepo) UpdateStatus(ctx context.Context, transactionID string, status model.PaymentStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE payments SET status = $1 WHERE transaction_id = $2`, status, transactionID)
	return err
}

func (r *PaymentRepo) FindByRegistration(ctx context.Context, registrationID string) (*model.Payment, error) {
	var p model.Payment
	err := r.pool.QueryRow(ctx,
		`SELECT id, registration_id, transaction_id, amount, provider, status, created_at
		 FROM payments WHERE registration_id = $1`, registrationID,
	).Scan(&p.ID, &p.RegistrationID, &p.TransactionID, &p.Amount, &p.Provider, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
func (r *PaymentRepo) FindAllPending(ctx context.Context) ([]model.Payment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, registration_id, transaction_id, amount, provider, status, created_at
		 FROM payments WHERE status = 'PENDING' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Payment
	for rows.Next() {
		var p model.Payment
		if err := rows.Scan(&p.ID, &p.RegistrationID, &p.TransactionID, &p.Amount, &p.Provider, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, nil
}

func (r *PaymentRepo) DeleteByRegistration(ctx context.Context, registrationID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM payments WHERE registration_id = $1`, registrationID)
	return err
}
