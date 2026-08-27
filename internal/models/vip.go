package models

import "time"

type VipSubscription struct {
	ID          int64     `db:"id"`
	UserID      int64     `db:"user_id"`
	Plan        string    `db:"plan"`
	StarsAmount int       `db:"stars_amount"`
	StartsAt    time.Time `db:"starts_at"`
	ExpiresAt   time.Time `db:"expires_at"`
}
