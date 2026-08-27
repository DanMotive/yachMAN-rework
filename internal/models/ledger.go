package models

import (
	"time"

	"github.com/google/uuid"
)

type LedgerEntry struct {
	ID           int64     `db:"id"`
	OperationID  uuid.UUID `db:"operation_id"`
	EntityType   string    `db:"entity_type"`
	EntityID     int64     `db:"entity_id"`
	Debit        int       `db:"debit"`
	Credit       int       `db:"credit"`
	BalanceAfter int       `db:"balance_after"`
	Description  string    `db:"description"`
	CreatedAt    time.Time `db:"created_at"`
}
