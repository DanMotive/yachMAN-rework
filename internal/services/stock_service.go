package services

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StockService struct {
	pool   *pgxpool.Pool
	ledger *LedgerService
}

func NewStockService(pool *pgxpool.Pool, ledger *LedgerService) *StockService {
	return &StockService{pool: pool, ledger: ledger}
}

// BuyShares creates a buy order and attempts to match with existing sell orders.
func (s *StockService) BuyShares(ctx context.Context, userID, corpID int64, amount int, pricePerShare int) error {
	if amount <= 0 {
		return fmt.Errorf("количество акций должно быть > 0")
	}
	if pricePerShare <= 0 {
		return fmt.Errorf("цена акции должна быть > 0")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	totalCost := amount * pricePerShare

	var balance int
	err = tx.QueryRow(ctx, `SELECT balance FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&balance)
	if err != nil {
		return fmt.Errorf("пользователь не найден")
	}
	if balance < totalCost {
		return fmt.Errorf("недостаточно средств: нужно %d ₽, есть %d ₽", totalCost, balance)
	}

	// Check not buying own corporation's shares
	var ownerID int64
	_ = tx.QueryRow(ctx, `SELECT owner_user_id FROM corporations WHERE id = $1`, corpID).Scan(&ownerID)
	if ownerID == userID {
		return fmt.Errorf("нельзя покупать акции своей корпорации")
	}

	opID := uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO share_orders (corporation_id, user_id, order_type, price_per_share, amount, operation_id)
		 VALUES ($1, $2, 'buy', $3, $4, $5)`,
		corpID, userID, pricePerShare, amount, opID)
	if err != nil {
		return fmt.Errorf("создание ордера: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Try to match after commit (matching uses its own transactions)
	s.matchOrders(ctx, corpID)
	return nil
}

// SellShares creates a sell order and attempts to match with existing buy orders.
func (s *StockService) SellShares(ctx context.Context, userID, corpID int64, amount int, pricePerShare int) error {
	if amount <= 0 {
		return fmt.Errorf("количество акций должно быть > 0")
	}
	if pricePerShare <= 0 {
		return fmt.Errorf("цена акции должна быть > 0")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var held int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(amount, 0) FROM shares WHERE corporation_id = $1 AND user_id = $2 FOR UPDATE`,
		corpID, userID).Scan(&held)
	if err != nil {
		held = 0
	}
	if held < amount {
		return fmt.Errorf("недостаточно акций: есть %d, продавать %d", held, amount)
	}

	opID := uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO share_orders (corporation_id, user_id, order_type, price_per_share, amount, operation_id)
		 VALUES ($1, $2, 'sell', $3, $4, $5)`,
		corpID, userID, pricePerShare, amount, opID)
	if err != nil {
		return fmt.Errorf("создание ордера: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	s.matchOrders(ctx, corpID)
	return nil
}

// matchOrders attempts to match open buy and sell orders for a corporation.
// Buys execute at the seller's asking price.
func (s *StockService) matchOrders(ctx context.Context, corpID int64) {
	// Get open sell orders (ascending price)
	sells, err := s.pool.Query(ctx,
		`SELECT id, user_id, price_per_share, amount, filled
		 FROM share_orders
		 WHERE corporation_id = $1 AND order_type = 'sell' AND status = 'open'
		 ORDER BY price_per_share ASC, created_at ASC`, corpID)
	if err != nil {
		return
	}
	defer sells.Close()

	for sells.Next() {
		var sellID int64
		var sellerID int64
		var sellPrice, sellAmount, sellFilled int
		if err := sells.Scan(&sellID, &sellerID, &sellPrice, &sellAmount, &sellFilled); err != nil {
			continue
		}
		sellRemaining := sellAmount - sellFilled
		if sellRemaining <= 0 {
			continue
		}

		// Find matching buy orders (descending price, price >= sell price)
		buys, err := s.pool.Query(ctx,
			`SELECT id, user_id, price_per_share, amount, filled
			 FROM share_orders
			 WHERE corporation_id = $1 AND order_type = 'buy' AND status = 'open'
			   AND price_per_share >= $2
			 ORDER BY price_per_share DESC, created_at ASC`, corpID, sellPrice)
		if err != nil {
			continue
		}

		for buys.Next() {
			var buyID int64
			var buyerID int64
			var buyPrice, buyAmount, buyFilled int
			if err := buys.Scan(&buyID, &buyerID, &buyPrice, &buyAmount, &buyFilled); err != nil {
				continue
			}
			buyRemaining := buyAmount - buyFilled
			if buyRemaining <= 0 {
				continue
			}
			if buyerID == sellerID {
				continue
			}

			// Execute trade at seller's price
			matchQty := sellRemaining
			if buyRemaining < matchQty {
				matchQty = buyRemaining
			}
			matchPrice := sellPrice // execute at seller's price
			totalCost := matchQty * matchPrice

			if err := s.executeTrade(ctx, corpID, buyerID, sellerID, matchQty, matchPrice, totalCost); err != nil {
				log.Printf("stock trade error: %v", err)
				continue
			}

			// Update fill amounts
			_, _ = s.pool.Exec(ctx, `UPDATE share_orders SET filled = filled + $1 WHERE id = $2`, matchQty, sellID)
			_, _ = s.pool.Exec(ctx, `UPDATE share_orders SET filled = filled + $1 WHERE id = $2`, matchQty, buyID)

			// Close fully filled orders
			_, _ = s.pool.Exec(ctx,
				`UPDATE share_orders SET status = 'filled' WHERE id = $1 AND filled >= amount`, sellID)
			_, _ = s.pool.Exec(ctx,
				`UPDATE share_orders SET status = 'filled' WHERE id = $1 AND filled >= amount`, buyID)

			sellRemaining -= matchQty
			if sellRemaining <= 0 {
				break
			}
		}
		buys.Close()

		if sellRemaining <= 0 {
			// Sell order fully filled
			_, _ = s.pool.Exec(ctx,
				`UPDATE share_orders SET status = 'filled' WHERE id = $1 AND filled >= amount`, sellID)
		}
	}
}

// executeTrade performs the atomic share + money transfer with ledger.
func (s *StockService) executeTrade(ctx context.Context, corpID, buyerID, sellerID int64, qty, price, totalCost int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Debit buyer
	var buyerBalance int
	err = tx.QueryRow(ctx, `SELECT balance FROM users WHERE id = $1 FOR UPDATE`, buyerID).Scan(&buyerBalance)
	if err != nil {
		return fmt.Errorf("buyer not found")
	}
	if buyerBalance < totalCost {
		return fmt.Errorf("buyer insufficient funds: has %d, need %d", buyerBalance, totalCost)
	}
	_, _ = tx.Exec(ctx, `UPDATE users SET balance = balance - $1 WHERE id = $2`, totalCost, buyerID)

	// 2. Credit seller
	_, _ = tx.Exec(ctx, `UPDATE users SET balance = balance + $1 WHERE id = $2`, totalCost, sellerID)

	// 3. Transfer shares: seller loses, buyer gains
	_, _ = tx.Exec(ctx,
		`UPDATE shares SET amount = amount - $1 WHERE corporation_id = $2 AND user_id = $3 AND amount >= $1`,
		qty, corpID, sellerID)
	_, _ = tx.Exec(ctx,
		`INSERT INTO shares (corporation_id, user_id, amount) VALUES ($1, $2, $3)
		 ON CONFLICT (corporation_id, user_id) DO UPDATE SET amount = shares.amount + $3`,
		corpID, buyerID, qty)

	// 4. Credit corporation treasury (management fee: 1% of trade value)
	fee := totalCost / 100
	if fee > 0 {
		_, _ = tx.Exec(ctx, `UPDATE corporations SET balance = balance + $1 WHERE id = $2`, fee, corpID)
	}

	// 5. Ledger entries
	opID := uuid.New()
	_, _ = tx.Exec(ctx,
		`INSERT INTO ledger_entries (operation_id, entity_type, entity_id, debit, credit, balance_after, description)
		 VALUES ($1, 'user', $2, $3, 0, 0, $4)`,
		opID, buyerID, totalCost, fmt.Sprintf("stock buy %d shares corp#%d @ %d₽", qty, corpID, price))

	opID2 := uuid.New()
	_, _ = tx.Exec(ctx,
		`INSERT INTO ledger_entries (operation_id, entity_type, entity_id, debit, credit, balance_after, description)
		 VALUES ($1, 'user', $2, 0, $3, 0, $4)`,
		opID2, sellerID, totalCost, fmt.Sprintf("stock sell %d shares corp#%d @ %d₽", qty, corpID, price))

	return tx.Commit(ctx)
}

// ExecuteOpenOrders is called by scheduler to process any unmatched orders.
func (s *StockService) ExecuteOpenOrders(ctx context.Context) (int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT corporation_id FROM share_orders WHERE status = 'open'`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var corpID int64
		if err := rows.Scan(&corpID); err != nil {
			continue
		}
		s.matchOrders(ctx, corpID)
		count++
	}
	return count, nil
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
