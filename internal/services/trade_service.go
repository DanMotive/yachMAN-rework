package services

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

type TradeService struct {
	pool *pgxpool.Pool
}

func NewTradeService(pool *pgxpool.Pool) *TradeService {
	return &TradeService{pool: pool}
}

func (s *TradeService) CreateContract(ctx context.Context, fromCityID, toCityID int64,
	resourceID string, qtyPerDay, pricePerUnit int, payers string, durationDays, penaltyPct int) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO trade_contracts (from_city_id, to_city_id, resource_id, quantity_per_day,
		 price_per_unit, payers, duration_days, penalty_pct, start_at, operation_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9)`,
		fromCityID, toCityID, resourceID, qtyPerDay, pricePerUnit, payers, durationDays, penaltyPct, uuid.New())
	return err
}

// ExecuteTradeContracts processes hourly trade contract execution.
func (s *TradeService) ExecuteTradeContracts(ctx context.Context) (int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, from_city_id, to_city_id, resource_id, quantity_per_day, price_per_unit, payers
		 FROM trade_contracts WHERE status = 'active' AND start_at IS NOT NULL`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var fromID, toID int64
		var resID string
		var qtyPerDay, pricePerUnit int
		var payers string
		if err := rows.Scan(&id, &fromID, &toID, &resID, &qtyPerDay, &pricePerUnit, &payers); err != nil {
			continue
		}
		hourlyQty := qtyPerDay / 24
		if hourlyQty < 1 {
			hourlyQty = 1
		}
		totalCost := hourlyQty * pricePerUnit

		tx, err := s.pool.Begin(ctx)
		if err != nil {
			continue
		}
		// Simplified: transfer resources and money between cities
		// Full implementation would check city resources and handle partial fulfillment
		_, _ = tx.Exec(ctx,
			`UPDATE city_resources SET stock = stock - $1 WHERE city_id = $2 AND resource_id = $3 AND stock >= $1`,
			hourlyQty, fromID, resID)
		_, _ = tx.Exec(ctx,
			`INSERT INTO city_resources (city_id, resource_id, stock) VALUES ($1, $2, $3)
			 ON CONFLICT (city_id, resource_id) DO UPDATE SET stock = city_resources.stock + $3`,
			toID, resID, hourlyQty)
		if payers == "from" {
			_, _ = tx.Exec(ctx, `UPDATE cities SET treasury = treasury - $1 WHERE id = $2 AND treasury >= $1`, totalCost, fromID)
		} else {
			_, _ = tx.Exec(ctx, `UPDATE cities SET treasury = treasury - $1 WHERE id = $2 AND treasury >= $1`, totalCost, toID)
		}
		_ = tx.Commit(ctx)
		count++
	}
	return count, nil
}

func (s *TradeService) GetCityContracts(ctx context.Context, cityID int64) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT tc.id, tc.resource_id, tc.quantity_per_day, tc.price_per_unit, tc.payers,
		        tc.duration_days, tc.status, tc.created_at,
		        cf.name as from_city, ct.name as to_city
		 FROM trade_contracts tc
		 JOIN cities cf ON tc.from_city_id = cf.id
		 JOIN cities ct ON tc.to_city_id = ct.id
		 WHERE tc.from_city_id = $1 OR tc.to_city_id = $1
		 ORDER BY tc.created_at DESC`, cityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contracts []map[string]interface{}
	for rows.Next() {
		var id int64
		var resID, payers, status, fromName, toName string
		var qtyPerDay, pricePerUnit, durationDays int
		var createdAt interface{}
		if err := rows.Scan(&id, &resID, &qtyPerDay, &pricePerUnit, &payers,
			&durationDays, &status, &createdAt, &fromName, &toName); err != nil {
			continue
		}
		contracts = append(contracts, map[string]interface{}{
			"id": id, "resource": resID, "qty_per_day": qtyPerDay,
			"price": pricePerUnit, "payers": payers, "from": fromName, "to": toName, "status": status,
		})
	}
	return contracts, nil
}
