package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"unihub-workshop/internal/model"
)

type NotificationRepo struct {
	pool *pgxpool.Pool
}

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{pool: pool}
}

func (r *NotificationRepo) Create(ctx context.Context, n *model.Notification) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO notifications (user_id, registration_id, channel, title, content, status, event_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (event_id, channel) DO NOTHING
		 RETURNING id, created_at`,
		n.UserID, n.RegistrationID, n.Channel, n.Title, n.Content, n.Status, n.EventID,
	).Scan(&n.ID, &n.CreatedAt)
}

func (r *NotificationRepo) UpdateStatus(ctx context.Context, id string, status model.NotificationStatus, errorMsg *string) error {
	if status == model.NotifSent {
		_, err := r.pool.Exec(ctx,
			`UPDATE notifications SET status = $1, sent_at = CURRENT_TIMESTAMP WHERE id = $2`, status, id)
		return err
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET status = $1, error_message = $2 WHERE id = $3`, status, errorMsg, id)
	return err
}

func (r *NotificationRepo) FindByUser(ctx context.Context, userID string) ([]model.Notification, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, registration_id, channel, title, content, status, event_id, error_message, created_at, sent_at
		 FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.RegistrationID, &n.Channel, &n.Title,
			&n.Content, &n.Status, &n.EventID, &n.ErrorMessage, &n.CreatedAt, &n.SentAt); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, nil
}
