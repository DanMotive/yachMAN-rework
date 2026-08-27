package services

import (
	"context"
	"math"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MarketService struct {
	pool *pgxpool.Pool
}

func NewMarketService(pool *pgxpool.Pool) *MarketService {
	return &MarketService{pool: pool}
}

// CalculatePrice computes resource price based on spec formula:
// price = base_price * clamp(0.50, 2.50, 1 + 0.60 * (demand - supply_index) / max(1, demand + supply_index))
// supply_index = stock + expected_production_24h * 0.25
func (s *MarketService) CalculatePrice(ctx context.Context, cityID int64, resourceID string) (int, error) {
	var basePrice, stock, demand int
	err := s.pool.QueryRow(ctx,
		`SELECT r.base_price, COALESCE(cr.stock, 0), COALESCE(cr.demand, 0)
		 FROM resources r
		 LEFT JOIN city_resources cr ON cr.resource_id = r.id AND cr.city_id = $1
		 WHERE r.id = $2`, cityID, resourceID).
		Scan(&basePrice, &stock, &demand)
	if err != nil {
		return 0, err
	}

	supplyIndex := float64(stock) // simplified: expected_production_24h * 0.25 ≈ 0 for now
	denom := math.Max(1, float64(demand)+supplyIndex)
	ratio := 1.0 + 0.60*(float64(demand)-supplyIndex)/denom
	clamped := math.Max(0.50, math.Min(2.50, ratio))
	price := int(math.Round(float64(basePrice) * clamped))
	if price < 1 {
		price = 1
	}
	return price, nil
}

// UpdateAllPrices recalculates and stores prices for all city_resources.
func (s *MarketService) UpdateAllPrices(ctx context.Context) error {
	rows, err := s.pool.Query(ctx,
		`SELECT cr.city_id, cr.resource_id, r.base_price, cr.stock, cr.demand
		 FROM city_resources cr JOIN resources r ON cr.resource_id = r.id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cityID int64
		var resourceID string
		var basePrice, stock, demand int
		if err := rows.Scan(&cityID, &resourceID, &basePrice, &stock, &demand); err != nil {
			continue
		}
		supplyIndex := float64(stock)
		denom := math.Max(1, float64(demand)+supplyIndex)
		ratio := 1.0 + 0.60*(float64(demand)-supplyIndex)/denom
		clamped := math.Max(0.50, math.Min(2.50, ratio))
		price := int(math.Round(float64(basePrice) * clamped))
		if price < 1 {
			price = 1
		}
		_, _ = s.pool.Exec(ctx,
			`UPDATE city_resources SET last_price = $1 WHERE city_id = $2 AND resource_id = $3`,
			price, cityID, resourceID)
	}
	return nil
}

func (s *MarketService) GetCityResources(ctx context.Context, cityID int64) (map[string]int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT resource_id, stock FROM city_resources WHERE city_id = $1`, cityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stock := make(map[string]int)
	for rows.Next() {
		var resID string
		var qty int
		if err := rows.Scan(&resID, &qty); err != nil {
			continue
		}
		stock[resID] = qty
	}
	return stock, nil
}
