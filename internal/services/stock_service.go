package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StockService struct {
	pool   *pgxpool.Pool
	ledger *LedgerService
}

func NewStockService(pool *pgxpool.Pool, ledger *LedgerService) *StockService {
	return &StockService{pool: pool, ledger: ledger}
}

func (s *StockService) BuyShares(ctx context.Context, userID, corpID int64, amount int, pricePerShare int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	totalCost := amount * pricePerShare

	var balance int
	err = tx.QueryRow(ctx, `SELECT balance FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&balance)
	if err != nil {
		return err
	}
	if balance < totalCost {
		return fmt.Errorf("недостаточно средств: %d ₽", totalCost)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO share_orders (corporation_id, user_id, order_type, price_per_share, amount, operation_id)
		 VALUES ($1, $2, 'buy', $3, $4, gen_random_uuid())`,
		corpID, userID, pricePerShare, amount)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *StockService) SellShares(ctx context.Context, userID, corpID int64, amount int, pricePerShare int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var held int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(amount, 0) FROM shares WHERE corporation_id = $1 AND user_id = $2`,
		corpID, userID).Scan(&held)
	if err != nil || held < amount {
		return fmt.Errorf("недостаточно акций: %d/%d", held, amount)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO share_orders (corporation_id, user_id, order_type, price_per_share, amount, operation_id)
		 VALUES ($1, $2, 'sell', $3, $4, gen_random_uuid())`,
		corpID, userID, pricePerShare, amount)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *StockService) GetUserShares(ctx context.Context, userID, corpID int64) (int, error) {
	var amount int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(amount, 0) FROM shares WHERE corporation_id = $1 AND user_id = $2`,
		corpID, userID).Scan(&amount)
	return amount, err
}

func (s *StockService) GetSharePrice(ctx context.Context, corpID int64) (int, error) {
	var totalShares, balance int
	err := s.pool.QueryRow(ctx,
		`SELECT total_shares, balance FROM corporations WHERE id = $1`, corpID).
		Scan(&totalShares, &balance)
	if err != nil {
		return 0, err
	}
	if totalShares == 0 {
		return 0, nil
	}
	return balance / totalShares, nil
}
