package models

import "time"

type Notification struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Title     string    `db:"title"`
	Body      string    `db:"body"`
	Read      bool      `db:"read"`
	CreatedAt time.Time `db:"created_at"`
}
