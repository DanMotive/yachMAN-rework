package models

import (
	"time"

	"github.com/google/uuid"
)

type WorkDefinition struct {
	ID              string `db:"id" json:"id"`
	Name            string `db:"name" json:"name"`
	Direction       string `db:"direction" json:"direction"`
	RequiredXP      int    `db:"required_xp" json:"required_xp"`
	DurationMinutes int    `db:"duration_minutes" json:"duration_minutes"`
	Payout          int    `db:"payout" json:"payout"`
	XPReward        int    `db:"xp_reward" json:"xp_reward"`
	ResourceType    string `db:"resource_type" json:"resource_type"`
	ResourceAmount  int    `db:"resource_amount" json:"resource_amount"`
}

type WorkRun struct {
	ID          int64      `db:"id" json:"id"`
	UserID      int64      `db:"user_id" json:"user_id"`
	WorkID      string     `db:"work_id" json:"work_id"`
	CityID      int64      `db:"city_id" json:"city_id"`
	StartedAt   time.Time  `db:"started_at" json:"started_at"`
	FinishesAt  time.Time  `db:"finishes_at" json:"finishes_at"`
	Completed   bool       `db:"completed" json:"completed"`
	OperationID uuid.UUID  `db:"operation_id" json:"operation_id"`
}
