package models

import "time"

type TelegramPayment struct {
	ID          int64     `db:"id"`
	UserID      int64     `db:"user_id"`
	StarsAmount int       `db:"stars_amount"`
	Payload     string    `db:"payload"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
}
