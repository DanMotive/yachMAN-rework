package services

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationService struct {
	pool *pgxpool.Pool
}

func NewNotificationService(pool *pgxpool.Pool) *NotificationService {
	return &NotificationService{pool: pool}
}

func (s *NotificationService) Create(ctx context.Context, userID int64, title, body string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO notifications (user_id, title, body) VALUES ($1, $2, $3)`,
		userID, title, body)
	return err
}

func (s *NotificationService) GetUnread(ctx context.Context, userID int64) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, title, body, created_at FROM notifications WHERE user_id = $1 AND read = FALSE ORDER BY created_at DESC LIMIT 20`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []map[string]interface{}
	for rows.Next() {
		var id int64
		var title, body string
		var createdAt interface{}
		if err := rows.Scan(&id, &title, &body, &createdAt); err != nil {
			continue
		}
		notifs = append(notifs, map[string]interface{}{
			"id": id, "title": title, "body": body, "created_at": createdAt,
		})
	}
	return notifs, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE notifications SET read = TRUE WHERE user_id = $1 AND read = FALSE`, userID)
	return err
}

// DeliverPending sends unsent notifications. Called by scheduler every minute.
func (s *NotificationService) DeliverPending(ctx context.Context) (int, error) {
	// In v1.0, notifications are stored in DB and read via /notifications.
	// Future: integrate Telegram Bot API sendMessage here.
	result, err := s.pool.Exec(ctx,
		`UPDATE notifications SET read = TRUE WHERE read = FALSE AND created_at < NOW() - INTERVAL '7 days'`)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}
