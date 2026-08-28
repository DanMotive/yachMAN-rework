package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentService struct {
	pool   *pgxpool.Pool
	ledger *LedgerService
}

func NewPaymentService(pool *pgxpool.Pool, ledger *LedgerService) *PaymentService {
	return &PaymentService{pool: pool, ledger: ledger}
nfunc (s *PaymentService) resolveInternalID(ctx context.Context, tx pgx.Tx, telegramID int64) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `SELECT id FROM users WHERE telegram_user_id = $1`, telegramID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("пользователь не найден")
	}
	return id, nil
}
}

func (s *PaymentService) Transfer(ctx context.Context, fromTGID int64, toTelegramID int64, amount int) error {
	if amount < 1 {
		return fmt.Errorf("минимальная сумма перевода: 1 ₽")
	}
	if amount > 10000000 {
		return fmt.Errorf("максимальная сумма перевода: 10 000 000 ₽")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	fromUserID, err := s.resolveInternalID(ctx, tx, fromTGID)
	if err != nil {
		return err
	}

	// Check recipient exists
	var toUserID int64
	err = tx.QueryRow(ctx,
		`SELECT id FROM users WHERE telegram_user_id = $1`, toTelegramID).
		Scan(&toUserID)
	if err != nil {
		return fmt.Errorf("получатель не найден")
	}

	if fromUserID == toUserID {
		return fmt.Errorf("нельзя переводить самому себе")
	}

	// Check daily limit (20 transfers/day)
	var todayCount int
	_ = tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM ledger_entries
		 WHERE entity_type = 'user' AND entity_id = $1
		   AND description LIKE 'transfer:%'
		   AND created_at >= CURRENT_DATE`, fromUserID).
		Scan(&todayCount)
	if todayCount >= 20 {
		return fmt.Errorf("достигнут дневной лимит переводов (20)")
	}

	// Debit sender
	if err := s.ledger.Debit(ctx, tx, "user", fromUserID, amount,
		fmt.Sprintf("transfer to %d", toUserID)); err != nil {
		return err
	}

	// Credit recipient
	if err := s.ledger.Credit(ctx, tx, "user", toUserID, amount,
		fmt.Sprintf("transfer from %d", fromUserID)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *PaymentService) GetDailyTransferCount(ctx context.Context, userID int64) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ledger_entries
		 WHERE entity_type = 'user' AND entity_id = $1
		   AND description LIKE 'transfer:%'
		   AND created_at >= CURRENT_DATE`, userID).Scan(&count)
	return count, err
}

func (s *PaymentService) GetUserByTelegramID(ctx context.Context, telegramUserID int64) (int64, error) {
	var userID int64
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM users WHERE telegram_user_id = $1`, telegramUserID).Scan(&userID)
	return userID, err
}

func (s *PaymentService) GetBalance(ctx context.Context, userID int64) (int, error) {
	var balance int
	err := s.pool.QueryRow(ctx,
		`SELECT balance FROM users WHERE id = $1`, userID).Scan(&balance)
	return balance, err
}

func (s *PaymentService) GetTransferHistory(ctx context.Context, userID int64, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx,
		`SELECT credit, debit, balance_after, description, created_at
		 FROM ledger_entries
		 WHERE entity_type = 'user' AND entity_id = $1
		   AND description LIKE 'transfer%'
		 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var credit, debit, balanceAfter int
		var description string
		var createdAt time.Time
		if err := rows.Scan(&credit, &debit, &balanceAfter, &description, &createdAt); err != nil {
			continue
		}
		history = append(history, map[string]interface{}{
			"credit": credit, "debit": debit,
			"balance_after": balanceAfter,
			"description":   description,
			"created_at":    createdAt,
		})
	}
	return history, nil
}
