package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LedgerService handles all atomic financial operations.
type LedgerService struct {
	pool *pgxpool.Pool
}

func NewLedgerService(pool *pgxpool.Pool) *LedgerService {
	return &LedgerService{pool: pool}
}

// Transfer moves money between two entities atomically.
// debitEntity is the source (money taken from), creditEntity is the destination.
func (s *LedgerService) Transfer(ctx context.Context, tx pgx.Tx,
	debitEntityType string, debitEntityID int64, debitAmount int,
	creditEntityType string, creditEntityID int64, creditAmount int,
	description string) error {

	opID := uuid.New()

	// Get current balance of debit entity
	var balance int
	var table, idCol string
	switch debitEntityType {
	case "user":
		table, idCol = "users", "id"
	case "city":
		table, idCol = "cities", "id"
	case "corporation":
		table, idCol = "corporations", "id"
	default:
		return fmt.Errorf("unknown entity type: %s", debitEntityType)
	}

	err := tx.QueryRow(ctx,
		fmt.Sprintf("SELECT balance FROM %s WHERE %s = $1 FOR UPDATE", table, idCol),
		debitEntityID).Scan(&balance)
	if err != nil {
		return fmt.Errorf("get %s balance: %w", debitEntityType, err)
	}

	if balance < debitAmount {
		return fmt.Errorf("insufficient funds: %s %d has %d, need %d",
			debitEntityType, debitEntityID, balance, debitAmount)
	}

	newBalance := balance - debitAmount

	// Update balance
	_, err = tx.Exec(ctx,
		fmt.Sprintf("UPDATE %s SET balance = $1 WHERE %s = $2", table, idCol),
		newBalance, debitEntityID)
	if err != nil {
		return fmt.Errorf("update %s balance: %w", debitEntityType, err)
	}

	// Record ledger entry for debit
	_, err = tx.Exec(ctx,
		`INSERT INTO ledger_entries (operation_id, entity_type, entity_id, debit, credit, balance_after, description)
		 VALUES ($1, $2, $3, $4, 0, $5, $6)`,
		opID, debitEntityType, debitEntityID, debitAmount, newBalance, description)
	if err != nil {
		return fmt.Errorf("insert debit ledger: %w", err)
	}

	// Credit to destination
	if creditAmount > 0 {
		var creditTable, creditIDCol string
		switch creditEntityType {
		case "user":
			creditTable, creditIDCol = "users", "id"
		case "city":
			creditTable, creditIDCol = "cities", "id"
		case "corporation":
			creditTable, creditIDCol = "corporations", "id"
		default:
			return fmt.Errorf("unknown credit entity type: %s", creditEntityType)
		}

		var creditBalance int
		err = tx.QueryRow(ctx,
			fmt.Sprintf("SELECT balance FROM %s WHERE %s = $1 FOR UPDATE", creditTable, creditIDCol),
			creditEntityID).Scan(&creditBalance)
		if err != nil {
			return fmt.Errorf("get %s balance: %w", creditEntityType, err)
		}

		newCreditBalance := creditBalance + creditAmount
		_, err = tx.Exec(ctx,
			fmt.Sprintf("UPDATE %s SET balance = $1 WHERE %s = $2", creditTable, creditIDCol),
			newCreditBalance, creditEntityID)
		if err != nil {
			return fmt.Errorf("update %s balance: %w", creditEntityType, err)
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO ledger_entries (operation_id, entity_type, entity_id, debit, credit, balance_after, description)
			 VALUES ($1, $2, $3, 0, $4, $5, $6)`,
			opID, creditEntityType, creditEntityID, creditAmount, newCreditBalance, description)
		if err != nil {
			return fmt.Errorf("insert credit ledger: %w", err)
		}
	}

	return nil
}

// Credit adds money to an entity and records the ledger entry.
func (s *LedgerService) Credit(ctx context.Context, tx pgx.Tx,
	entityType string, entityID int64, amount int, description string) error {

	opID := uuid.New()
	var table, idCol string
	switch entityType {
	case "user":
		table, idCol = "users", "id"
	case "city":
		table, idCol = "cities", "id"
	case "corporation":
		table, idCol = "corporations", "id"
	default:
		return fmt.Errorf("unknown entity type: %s", entityType)
	}

	var balance int
	err := tx.QueryRow(ctx,
		fmt.Sprintf("SELECT balance FROM %s WHERE %s = $1 FOR UPDATE", table, idCol),
		entityID).Scan(&balance)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	newBalance := balance + amount
	_, err = tx.Exec(ctx,
		fmt.Sprintf("UPDATE %s SET balance = $1 WHERE %s = $2", table, idCol),
		newBalance, entityID)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO ledger_entries (operation_id, entity_type, entity_id, debit, credit, balance_after, description)
		 VALUES ($1, $2, $3, 0, $4, $5, $6)`,
		opID, entityType, entityID, amount, newBalance, description)
	if err != nil {
		return fmt.Errorf("insert ledger: %w", err)
	}

	return nil
}

// Debit removes money from an entity and records the ledger entry.
func (s *LedgerService) Debit(ctx context.Context, tx pgx.Tx,
	entityType string, entityID int64, amount int, description string) error {

	opID := uuid.New()
	var table, idCol string
	switch entityType {
	case "user":
		table, idCol = "users", "id"
	case "city":
		table, idCol = "cities", "id"
	case "corporation":
		table, idCol = "corporations", "id"
	default:
		return fmt.Errorf("unknown entity type: %s", entityType)
	}

	var balance int
	err := tx.QueryRow(ctx,
		fmt.Sprintf("SELECT balance FROM %s WHERE %s = $1 FOR UPDATE", table, idCol),
		entityID).Scan(&balance)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	if balance < amount {
		return fmt.Errorf("insufficient funds: has %d, need %d", balance, amount)
	}

	newBalance := balance - amount
	_, err = tx.Exec(ctx,
		fmt.Sprintf("UPDATE %s SET balance = $1 WHERE %s = $2", table, idCol),
		newBalance, entityID)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO ledger_entries (operation_id, entity_type, entity_id, debit, credit, balance_after, description)
		 VALUES ($1, $2, $3, $4, 0, $5, $6)`,
		opID, entityType, entityID, amount, newBalance, description)
	if err != nil {
		return fmt.Errorf("insert ledger: %w", err)
	}

	return nil
}
