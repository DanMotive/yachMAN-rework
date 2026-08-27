package models

import "time"

type Event struct {
	ID          int64     `db:"id"`
	Type        string    `db:"type"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CityID      *int64    `db:"city_id"`
	StartAt     time.Time `db:"start_at"`
	EndAt       time.Time `db:"end_at"`
	CreatedAt   time.Time `db:"created_at"`
}

type EventEffect struct {
	ID         int64   `db:"id"`
	EventID    int64   `db:"event_id"`
	TargetType string  `db:"target_type"`
	TargetID   *string `db:"target_id"`
	Modifier   float64 `db:"modifier"`
}
