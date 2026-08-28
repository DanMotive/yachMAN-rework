package services

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
// Uses FOR UPDATE SKIP LOCKED to prevent concurrent matching on the same corporation.
func (s *StockService) matchOrders(ctx context.Context, corpID int64) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)

	// Lock all open orders for this corporation to prevent concurrent matching
	rows, err := tx.Query(ctx,
		`SELECT id, user_id, order_type, price_per_share, amount, filled
		 FROM share_orders
		 WHERE corporation_id = $1 AND status = 'open'
		 ORDER BY order_type DESC, price_per_share ASC, created_at ASC
		 FOR UPDATE SKIP LOCKED`, corpID)
	if err != nil {
		return
	}

	// Collect all orders into memory (small result set)
	type order struct {
		id, userID, price, amount, filled int64
		isSell                             bool
	}
	var allOrders []order
	for rows.Next() {
		var o order
		var otype string
		if err := rows.Scan(&o.id, &o.userID, &otype, &o.price, &o.amount, &o.filled); err != nil {
			continue
		}
		o.isSell = otype == "sell"
		allOrders = append(allOrders, o)
	}
	rows.Close()

	if len(allOrders) == 0 {
		return
	}

	// Separate into sells and buys
	var sells, buys []order
	for _, o := range allOrders {
		if o.isSell {
			sells = append(sells, o)
		} else {
			buys = append(buys, o)
		}
	}

	// Match: for each sell order, find matching buy orders
	for si := range sells {
		sell := &sells[si]
		sellRemaining := sell.amount - sell.filled
		if sellRemaining <= 0 {
			continue
		}

		for bi := range buys {
			buy := &buys[bi]
			buyRemaining := buy.amount - buy.filled
			if buyRemaining <= 0 {
				continue
			}
			if buy.userID == sell.userID {
				continue
			}
			if buy.price < sell.price {
				continue // buy price too low
			}

			matchQty := sellRemaining
			if buyRemaining < matchQty {
				matchQty = buyRemaining
			}
			matchPrice := sell.price
			totalCost := int(matchQty) * int(matchPrice)

			if err := s.executeTradeTx(ctx, tx, corpID, buy.userID, sell.userID,
				int(matchQty), int(matchPrice), totalCost); err != nil {
				log.Printf("stock trade error corp#%d: %v", corpID, err)
				continue
			}

			// Update fill amounts within the same transaction
			_, _ = tx.Exec(ctx, `UPDATE share_orders SET filled = filled + $1 WHERE id = $2`, matchQty, sell.id)
			_, _ = tx.Exec(ctx, `UPDATE share_orders SET filled = filled + $1 WHERE id = $2`, matchQty, buy.id)

			// Close fully filled orders
			_, _ = tx.Exec(ctx,
				`UPDATE share_orders SET status = 'filled' WHERE id = $1 AND filled >= amount`, sell.id)
			_, _ = tx.Exec(ctx,
				`UPDATE share_orders SET status = 'filled' WHERE id = $1 AND filled >= amount`, buy.id)

			sell.filled += matchQty
			sellRemaining -= matchQty
			buy.filled += matchQty
			if sellRemaining <= 0 {
				break
			}
		}
	}

	// Commit all changes atomically
	if err := tx.Commit(ctx); err != nil {
		log.Printf("stock matching commit error corp#%d: %v", corpID, err)
	}
}

// executeTradeTx performs the atomic share + money transfer within an existing transaction.
func (s *StockService) executeTradeTx(ctx context.Context, tx pgx.Tx,
	corpID, buyerID, sellerID int64, qty, price, totalCost int) error {

	// 1. Debit buyer
	var buyerBalance int
	err := tx.QueryRow(ctx, `SELECT balance FROM users WHERE id = $1 FOR UPDATE`, buyerID).Scan(&buyerBalance)
	if err != nil {
		return fmt.Errorf("buyer not found: %w", err)
	}
	if buyerBalance < totalCost {
		return fmt.Errorf("buyer insufficient funds: has %d, need %d", buyerBalance, totalCost)
	}
	_, err = tx.Exec(ctx, `UPDATE users SET balance = balance - $1 WHERE id = $2`, totalCost, buyerID)
	if err != nil {
		return fmt.Errorf("debit buyer: %w", err)
	}

	// 2. Credit seller
	_, err = tx.Exec(ctx, `UPDATE users SET balance = balance + $1 WHERE id = $2`, totalCost, sellerID)
	if err != nil {
		return fmt.Errorf("credit seller: %w", err)
	}

	// 3. Transfer shares: seller loses, buyer gains
	_, err = tx.Exec(ctx,
		`UPDATE shares SET amount = amount - $1 WHERE corporation_id = $2 AND user_id = $3 AND amount >= $1`,
		qty, corpID, sellerID)
	if err != nil {
		return fmt.Errorf("transfer shares from seller: %w", err)
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO shares (corporation_id, user_id, amount) VALUES ($1, $2, $3)
		 ON CONFLICT (corporation_id, user_id) DO UPDATE SET amount = shares.amount + $3`,
		corpID, buyerID, qty)
	if err != nil {
		return fmt.Errorf("transfer shares to buyer: %w", err)
	}

	// 4. Credit corporation treasury (management fee: 1% of trade value)
	fee := totalCost / 100
	if fee > 0 {
		_, err = tx.Exec(ctx, `UPDATE corporations SET balance = balance + $1 WHERE id = $2`, fee, corpID)
		if err != nil {
			return fmt.Errorf("credit corp fee: %w", err)
		}
	}

	// 5. Ledger entries with actual balance_after
	var buyerAfter, sellerAfter int

	// 5. Ledger entries with actual balance_after
	var buyerAfter, sellerAfter int
	_ = tx.QueryRow(ctx, `SELECT balance FROM users WHERE id = $1`, buyerID).Scan(&buyerAfter)
	_ = tx.QueryRow(ctx, `SELECT balance FROM users WHERE id = $1`, sellerID).Scan(&sellerAfter)

	opID := uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO ledger_entries (operation_id, entity_type, entity_id, debit, credit, balance_after, description)
		 VALUES ($1, 'user', $2, $3, 0, $4, $5)`,
		opID, buyerID, totalCost, buyerAfter,
		fmt.Sprintf("stock buy %d shares corp#%d @ %d₽", qty, corpID, price))
	if err != nil {
		return fmt.Errorf("ledger buyer: %w", err)
	}

	opID2 := uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO ledger_entries (operation_id, entity_type, entity_id, debit, credit, balance_after, description)
		 VALUES ($1, 'user', $2, 0, $3, $4, $5)`,
		opID2, sellerID, totalCost, sellerAfter,
		fmt.Sprintf("stock sell %d shares corp#%d @ %d₽", qty, corpID, price))
	if err != nil {
		return fmt.Errorf("ledger seller: %w", err)
	}

	return nil
}

// ExecuteOpenOrders is called by scheduler to process any unmatched orders.
func (s *StockService) ExecuteOpenOrders(ctx context.Context) (int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT corporation_id FROM share_orders WHERE status = 'open'
		 FOR UPDATE SKIP LOCKED`)
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
