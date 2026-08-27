package models

import (
	"time"

	"github.com/google/uuid"
)

type TradeContract struct {
	ID              int64      `db:"id"`
	FromCityID      int64      `db:"from_city_id"`
	ToCityID        int64      `db:"to_city_id"`
	ResourceID      string     `db:"resource_id"`
	QuantityPerDay  int        `db:"quantity_per_day"`
	PricePerUnit    int        `db:"price_per_unit"`
	Payers          string     `db:"payers"`
	DurationDays    int        `db:"duration_days"`
	StartAt         *time.Time `db:"start_at"`
	PenaltyPct      int        `db:"penalty_pct"`
	Status          string     `db:"status"`
	CreatedAt       time.Time  `db:"created_at"`
	OperationID     uuid.UUID  `db:"operation_id"`
}
