package models

import (
	"time"

	"github.com/google/uuid"
)

type WorkDefinition struct {
	ID               string `db:"id"`
	Name             string `db:"name"`
	Direction        string `db:"direction"`
	RequiredXP       int    `db:"required_xp"`
	DurationMinutes  int    `db:"duration_minutes"`
	Payout           int    `db:"payout"`
	XPReward         int    `db:"xp_reward"`
	ResourceType     string `db:"resource_type"`
	ResourceAmount   int    `db:"resource_amount"`
}

type WorkRun struct {
	ID         int64      `db:"id"`
	UserID     int64      `db:"user_id"`
	WorkID     string     `db:"work_id"`
	CityID     int64      `db:"city_id"`
	StartedAt  time.Time  `db:"started_at"`
	FinishesAt time.Time  `db:"finishes_at"`
	Completed  bool       `db:"completed"`
	OperationID uuid.UUID `db:"operation_id"`
}
