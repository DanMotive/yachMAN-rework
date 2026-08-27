package models

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID         int64           `db:"id"`
	UserID     int64           `db:"user_id"`
	Action     string          `db:"action"`
	EntityType *string         `db:"entity_type"`
	EntityID   *int64          `db:"entity_id"`
	Details    json.RawMessage `db:"details"`
	CreatedAt  time.Time       `db:"created_at"`
}
