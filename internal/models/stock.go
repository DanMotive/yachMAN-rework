package models

import (
	"time"

	"github.com/google/uuid"
)

type Share struct {
	ID             int64 `db:"id"`
	CorporationID  int64 `db:"corporation_id"`
	UserID         int64 `db:"user_id"`
	Amount         int   `db:"amount"`
}

type ShareOrder struct {
	ID             int64      `db:"id"`
	CorporationID  int64      `db:"corporation_id"`
	UserID         int64      `db:"user_id"`
	OrderType      string     `db:"order_type"`
	PricePerShare  int        `db:"price_per_share"`
	Amount         int        `db:"amount"`
	Filled         int        `db:"filled"`
	Status         string     `db:"status"`
	CreatedAt      time.Time  `db:"created_at"`
	OperationID    uuid.UUID  `db:"operation_id"`
}
