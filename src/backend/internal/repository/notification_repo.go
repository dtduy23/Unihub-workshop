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
		`INSERT INTO notifications (user_id, registration_id, channel, title, content, status, event_id,link)
		 VALUES ($1, $2, $3, $4, $5, $6, $7,$8)
		 ON CONFLICT (event_id, channel) DO NOTHING
		 RETURNING id, created_at`,
		n.UserID, n.RegistrationID, n.Channel, n.Title, n.Content, n.Status, n.EventID, n.Link,
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

func (r *NotificationRepo) PrepareEmail(ctx context.Context, n *model.Notification) error {
	return r.pool.QueryRow(ctx, `INSERT INTO notifications(user_id,registration_id,channel,title,content,status,event_id,link) VALUES($1,$2,'EMAIL',$3,$4,'PENDING',$5,$6)
 ON CONFLICT(event_id,channel) DO UPDATE SET event_id=EXCLUDED.event_id RETURNING id,status`, n.UserID, n.RegistrationID, n.Title, n.Content, n.EventID, n.Link).Scan(&n.ID, &n.Status)
}

func (r *NotificationRepo) FindByUser(ctx context.Context, userID string) ([]model.Notification, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, registration_id, channel, title, content, status, event_id, error_message, created_at, sent_at,link
		 FROM notifications WHERE user_id = $1 AND channel='WEB' ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifs := []model.Notification{}
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.RegistrationID, &n.Channel, &n.Title,
			&n.Content, &n.Status, &n.EventID, &n.ErrorMessage, &n.CreatedAt, &n.SentAt, &n.Link); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, nil
}
