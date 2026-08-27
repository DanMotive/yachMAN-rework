package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"yachman/internal/models"
)

type CityService struct {
	pool *pgxpool.Pool
}

func NewCityService(pool *pgxpool.Pool) *CityService {
	return &CityService{pool: pool}
}

func (s *CityService) GetCityByChatID(ctx context.Context, chatID int64) (*models.City, error) {
	var c models.City
	err := s.pool.QueryRow(ctx,
		`SELECT id, chat_id, name, description, level, npc_population, development_points,
		        treasury, tax_rate_business, tax_rate_corporate, tax_rate_income,
		        last_tax_change_at, access_mode, mayor_user_id, public_listing, created_at
		 FROM cities WHERE chat_id = $1`, chatID).Scan(
		&c.ID, &c.ChatID, &c.Name, &c.Description, &c.Level, &c.NPCPopulation,
		&c.DevelopmentPoints, &c.Treasury, &c.TaxRateBusiness, &c.TaxRateCorporate,
		&c.TaxRateIncome, &c.LastTaxChangeAt, &c.AccessMode, &c.MayorUserID,
		&c.PublicListing, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CityService) GetCityByID(ctx context.Context, cityID int64) (*models.City, error) {
	var c models.City
	err := s.pool.QueryRow(ctx,
		`SELECT id, chat_id, name, description, level, npc_population, development_points,
		        treasury, tax_rate_business, tax_rate_corporate, tax_rate_income,
		        last_tax_change_at, access_mode, mayor_user_id, public_listing, created_at
		 FROM cities WHERE id = $1`, cityID).Scan(
		&c.ID, &c.ChatID, &c.Name, &c.Description, &c.Level, &c.NPCPopulation,
		&c.DevelopmentPoints, &c.Treasury, &c.TaxRateBusiness, &c.TaxRateCorporate,
		&c.TaxRateIncome, &c.LastTaxChangeAt, &c.AccessMode, &c.MayorUserID,
		&c.PublicListing, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// RegisterCity creates a new city from a Telegram group.
func (s *CityService) RegisterCity(ctx context.Context, chatID int64, name string, mayorUserID int64) error {
	return s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		// Check chat not already registered
		var count int
		err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM cities WHERE chat_id = $1`, chatID).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("город уже зарегистрирован")
		}

		// Create city
		var cityID int64
		err = tx.QueryRow(ctx,
			`INSERT INTO cities (chat_id, name, mayor_user_id) VALUES ($1, $2, $3) RETURNING id`,
			chatID, name, mayorUserID).Scan(&cityID)
		if err != nil {
			return err
		}

		// Add mayor as member and admin
		_, err = tx.Exec(ctx,
			`INSERT INTO city_members (city_id, user_id, role) VALUES ($1, $2, 'mayor')`,
			cityID, mayorUserID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO city_admins (city_id, user_id, position) VALUES ($1, $2, 'mayor')`,
			cityID, mayorUserID)
		if err != nil {
			return err
		}

		// Set user's city
		_, err = tx.Exec(ctx, `UPDATE users SET city_id = $1 WHERE id = $2`, cityID, mayorUserID)
		return err
	})
}

// JoinCity adds a player to a city.
func (s *CityService) JoinCity(ctx context.Context, userID int64, cityID int64) error {
	return s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		// Check user has no city
		var currentCity *int64
		err := tx.QueryRow(ctx, `SELECT city_id FROM users WHERE id = $1`, userID).Scan(&currentCity)
		if err != nil {
			return err
		}
		if currentCity != nil {
			return fmt.Errorf("вы уже состоите в городе. Сначала покиньте текущий город")
		}

		// Check city access mode
		var accessMode string
		err = tx.QueryRow(ctx, `SELECT access_mode FROM cities WHERE id = $1 FOR UPDATE`, cityID).
			Scan(&accessMode)
		if err != nil {
			return fmt.Errorf("город не найден")
		}
		if accessMode == "closed" {
			return fmt.Errorf("город закрыт для вступления")
		}

		// Add member
		_, err = tx.Exec(ctx,
			`INSERT INTO city_members (city_id, user_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`, cityID, userID)
		if err != nil {
			return err
		}

		// Update user's city
		_, err = tx.Exec(ctx, `UPDATE users SET city_id = $1 WHERE id = $2`, cityID, userID)
		return err
	})
}

// LeaveCity removes a player from their city.
func (s *CityService) LeaveCity(ctx context.Context, userID int64) error {
	return s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var cityID *int64
		err := tx.QueryRow(ctx, `SELECT city_id FROM users WHERE id = $1 FOR UPDATE`, userID).
			Scan(&cityID)
		if err != nil {
			return err
		}
		if cityID == nil {
			return fmt.Errorf("вы не состоите в городе")
		}

		// Remove from city_members
		_, _ = tx.Exec(ctx,
			`DELETE FROM city_members WHERE city_id = $1 AND user_id = $2`, *cityID, userID)
		_, _ = tx.Exec(ctx,
			`DELETE FROM city_admins WHERE city_id = $1 AND user_id = $2`, *cityID, userID)

		// Update user
		_, err = tx.Exec(ctx, `UPDATE users SET city_id = NULL WHERE id = $1`, userID)
		return err
	})
}

func (s *CityService) ListPublicCities(ctx context.Context) ([]models.City, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, chat_id, name, description, level, npc_population, development_points,
		        treasury, tax_rate_business, tax_rate_corporate, tax_rate_income,
		        last_tax_change_at, access_mode, mayor_user_id, public_listing, created_at
		 FROM cities WHERE public_listing = TRUE ORDER BY npc_population DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cities []models.City
	for rows.Next() {
		var c models.City
		if err := rows.Scan(&c.ID, &c.ChatID, &c.Name, &c.Description, &c.Level, &c.NPCPopulation,
			&c.DevelopmentPoints, &c.Treasury, &c.TaxRateBusiness, &c.TaxRateCorporate,
			&c.TaxRateIncome, &c.LastTaxChangeAt, &c.AccessMode, &c.MayorUserID,
			&c.PublicListing, &c.CreatedAt); err != nil {
			return nil, err
		}
		cities = append(cities, c)
	}
	return cities, nil
}

func (s *CityService) GetPlayerCount(ctx context.Context, cityID int64) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM city_members WHERE city_id = $1`, cityID).Scan(&count)
	return count, err
}

func (s *CityService) IsAdmin(ctx context.Context, userID int64, cityID int64) (bool, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM city_admins WHERE city_id = $1 AND user_id = $2`,
		cityID, userID).Scan(&count)
	return count > 0, err
}
