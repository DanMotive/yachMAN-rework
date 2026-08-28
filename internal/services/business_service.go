package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BusinessService struct {
	pool   *pgxpool.Pool
	ledger *LedgerService
}

func NewBusinessService(pool *pgxpool.Pool, ledger *LedgerService) *BusinessService {
	return &BusinessService{pool: pool, ledger: ledger}
}

// ProcessBusinessTicks runs one production tick for all businesses that are due.
// Called hourly by scheduler.
func (s *BusinessService) ProcessBusinessTicks(ctx context.Context) (int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT b.id, b.city_id, b.type_id, b.power_pct, b.corporation_id,
		        bt.input_a_resource, bt.input_a_amount, bt.input_b_resource, bt.input_b_amount,
		        bt.output_resource, bt.output_amount, bt.npc_staff
		 FROM businesses b
		 JOIN business_types bt ON b.type_id = bt.type_id
		 WHERE b.last_tick_at IS NULL OR b.last_tick_at <= NOW() - INTERVAL '60 minutes'`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var bizID, cityID int64
		var typeID string
		var powerPct int
		var corpID *int64
		var inARes, inBRes, outRes string
		var inAAmt, inBAmt, outAmt, npcStaff int

		if err := rows.Scan(&bizID, &cityID, &typeID, &powerPct, &corpID,
			&inARes, &inAAmt, &inBRes, &inBAmt, &outRes, &outAmt, &npcStaff); err != nil {
			continue
		}

		s.processOneBusiness(ctx, bizID, cityID, powerPct, corpID,
			inARes, inAAmt, inBRes, inBAmt, outRes, outAmt, npcStaff)
		count++
	}
	return count, nil
}

func (s *BusinessService) processOneBusiness(ctx context.Context, bizID, cityID int64, powerPct int,
	corpID *int64, inARes string, inAAmt int, inBRes string, inBAmt int, outRes string, outAmt int, npcStaff int) {

	factor := float64(powerPct) / 100.0

	// Get city stock
	var stockA, stockB int
	_ = s.pool.QueryRow(ctx,
		`SELECT COALESCE(stock,0) FROM city_resources WHERE city_id=$1 AND resource_id=$2`,
		cityID, inARes).Scan(&stockA)
	_ = s.pool.QueryRow(ctx,
		`SELECT COALESCE(stock,0) FROM city_resources WHERE city_id=$1 AND resource_id=$2`,
		cityID, inBRes).Scan(&stockB)

	// Calculate available production
	needA := int(float64(inAAmt) * factor)
	needB := int(float64(inBAmt) * factor)

	prodFactor := 1.0
	if needA > 0 && stockA < needA {
		prodFactor = float64(stockA) / float64(needA)
	}
	if needB > 0 && stockB < needB {
		f2 := float64(stockB) / float64(needB)
		if f2 < prodFactor {
			prodFactor = f2
		}
	}

	consumedA := int(float64(inAAmt) * factor * prodFactor)
	consumedB := int(float64(inBAmt) * factor * prodFactor)
	produced := int(float64(outAmt) * factor * prodFactor)

	if produced <= 0 {
		// Still update last_tick_at
		_, _ = s.pool.Exec(ctx, `UPDATE businesses SET last_tick_at = NOW() WHERE id = $1`, bizID)
		return
	}

	// Deduct inputs, add outputs in a transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)

	_, _ = tx.Exec(ctx,
		`UPDATE city_resources SET stock = stock - $1 WHERE city_id = $2 AND resource_id = $3 AND stock >= $1`,
		consumedA, cityID, inARes)
	_, _ = tx.Exec(ctx,
		`UPDATE city_resources SET stock = stock - $1 WHERE city_id = $2 AND resource_id = $3 AND stock >= $1`,
		consumedB, cityID, inBRes)
	_, _ = tx.Exec(ctx,
		`INSERT INTO city_resources (city_id, resource_id, stock) VALUES ($1, $2, $3)
		 ON CONFLICT (city_id, resource_id) DO UPDATE SET stock = city_resources.stock + $3`,
		cityID, outRes, produced)

	// Pay corporate tax if owned by corporation
	if corpID != nil {
		var taxRate float64
		_ = s.pool.QueryRow(ctx, `SELECT tax_rate_corporate FROM cities WHERE id = $1`, cityID).Scan(&taxRate)
		taxAmount := int(float64(produced) * taxRate / 100.0)
		if taxAmount > 0 {
			_, _ = tx.Exec(ctx, `UPDATE city_resources SET stock = stock - $1 WHERE city_id = $2 AND resource_id = $3 AND stock >= $1`,
				taxAmount, cityID, outRes)
			_, _ = tx.Exec(ctx, `UPDATE cities SET treasury = treasury + $1 WHERE id = $2`, taxAmount, cityID)
		}
	}

	_, _ = tx.Exec(ctx, `UPDATE businesses SET last_tick_at = NOW() WHERE id = $1`, bizID)
	_ = tx.Commit(ctx)
}

// ListUserBusinesses returns businesses owned by a user.
func (s *BusinessService) ListUserBusinesses(ctx context.Context, userID int64) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT b.id, b.name, bt.name as type_name, b.power_pct, b.city_id,
		        bt.input_a_resource, bt.input_a_amount, bt.input_b_resource, bt.input_b_amount,
		        bt.output_resource, bt.output_amount, bt.npc_staff, c.name as city_name
		 FROM businesses b
		 JOIN business_types bt ON b.type_id = bt.type_id
		 LEFT JOIN cities c ON b.city_id = c.id
		 WHERE b.owner_user_id = $1 OR b.corporation_id IN (
		     SELECT id FROM corporations WHERE owner_user_id = $1
		 )
		 ORDER BY b.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]interface{}
	for rows.Next() {
		var id, powerPct, inAAmt, inBAmt, outAmt, npcStaff int
		var cityID int64
		var name, typeName, inARes, inBRes, outRes, cityName string
		if err := rows.Scan(&id, &name, &typeName, &powerPct, &cityID,
			&inARes, &inAAmt, &inBRes, &inBAmt, &outRes, &outAmt, &npcStaff, &cityName); err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"id": id, "name": name, "type": typeName, "power": powerPct,
			"city": cityName, "input_a": fmt.Sprintf("%s %d", inARes, inAAmt),
			"input_b": fmt.Sprintf("%s %d", inBRes, inBAmt),
			"output": fmt.Sprintf("%s %d", outRes, outAmt), "npc_staff": npcStaff,
		})
	}
	return result, nil
}
