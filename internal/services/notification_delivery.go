package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationDelivery struct {
	pool      *pgxpool.Pool
	botToken  string
}

func NewNotificationDelivery(pool *pgxpool.Pool, botToken string) *NotificationDelivery {
	return &NotificationDelivery{pool: pool, botToken: botToken}
}

// DeliverPendingNotifications sends unread notifications via Telegram Bot API.
// Called by scheduler every minute.
func (d *NotificationDelivery) DeliverPendingNotifications(ctx context.Context) (int, error) {
	if d.botToken == "" {
		return 0, nil
	}

	rows, err := d.pool.Query(ctx,
		`SELECT n.id, n.user_id, n.title, n.body, u.telegram_user_id
		 FROM notifications n
		 JOIN users u ON n.user_id = u.id
		 WHERE n.read = FALSE AND n.delivered = FALSE
		 ORDER BY n.created_at ASC
		 LIMIT 50`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, userID int64
		var title, body string
		var telegramID int64
		if err := rows.Scan(&id, &userID, &title, &body, &telegramID); err != nil {
			continue
		}

		text := fmt.Sprintf("📬 %s\n\n%s", title, body)
		if d.sendMessage(telegramID, text) {
			// Mark as delivered
			_, _ = d.pool.Exec(ctx,
				`UPDATE notifications SET delivered = TRUE WHERE id = $1`, id)
			count++
		}
	}
	return count, nil
}

func (d *NotificationDelivery) sendMessage(chatID int64, text string) bool {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", d.botToken)
	data := url.Values{
		"chat_id": {strconv.FormatInt(chatID, 10)},
		"text":    {text},
	}
	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		log.Printf("notification delivery error: %v", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// CleanupOldUpdates removes processed updates older than 7 days.
func (d *NotificationDelivery) CleanupOldUpdates(ctx context.Context) (int, error) {
	result, err := d.pool.Exec(ctx,
		`DELETE FROM processed_updates WHERE processed_at < NOW() - INTERVAL '7 days'`)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}
