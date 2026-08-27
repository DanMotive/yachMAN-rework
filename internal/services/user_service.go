package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"yachman/internal/models"
)

type UserService struct {
	pool   *pgxpool.Pool
	ledger *LedgerService
}

func NewUserService(pool *pgxpool.Pool, ledger *LedgerService) *UserService {
	return &UserService{pool: pool, ledger: ledger}
}

// GetOrCreateUser finds existing user or creates a new one.
func (s *UserService) GetOrCreateUser(ctx context.Context, telegramUserID int64) (*models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, telegram_user_id, balance, global_level, global_xp, city_id, active_job,
		        corporation_id, corporation_role, vip_until, daily_streak, last_daily_at,
		        last_active_at, created_at
		 FROM users WHERE telegram_user_id = $1`, telegramUserID).Scan(
		&u.ID, &u.TelegramUserID, &u.Balance, &u.GlobalLevel, &u.GlobalXP,
		&u.CityID, &u.ActiveJob, &u.CorporationID, &u.CorporationRole,
		&u.VipUntil, &u.DailyStreak, &u.LastDailyAt, &u.LastActiveAt, &u.CreatedAt)

	if err == pgx.ErrNoRows {
		// Create new user with 1000 starting balance
		err = s.pool.QueryRow(ctx,
			`INSERT INTO users (telegram_user_id, balance) VALUES ($1, 1000)
			 RETURNING id, telegram_user_id, balance, global_level, global_xp, city_id, active_job,
			           corporation_id, corporation_role, vip_until, daily_streak, last_daily_at,
			           last_active_at, created_at`, telegramUserID).Scan(
			&u.ID, &u.TelegramUserID, &u.Balance, &u.GlobalLevel, &u.GlobalXP,
			&u.CityID, &u.ActiveJob, &u.CorporationID, &u.CorporationRole,
			&u.VipUntil, &u.DailyStreak, &u.LastDailyAt, &u.LastActiveAt, &u.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}

		// Initialize all skill directions at 0
		for _, dir := range []string{"добыча", "лес", "топливо", "энергетика", "металлургия",
			"строительство", "химия", "IT", "торговля", "агро",
			"транспорт", "питание", "ремонт", "медицина", "образование",
			"наука", "безопасность", "медиа", "коммунальные услуги", "переработка"} {
			_, _ = s.pool.Exec(ctx,
				`INSERT INTO user_skills (user_id, direction, xp) VALUES ($1, $2, 0) ON CONFLICT DO NOTHING`,
				u.ID, dir)
		}
		return &u, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Update last_active_at
	_, _ = s.pool.Exec(ctx, "UPDATE users SET last_active_at = NOW() WHERE id = $1", u.ID)
	return &u, nil
}

func (s *UserService) GetUserByTGID(ctx context.Context, telegramUserID int64) (*models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, telegram_user_id, balance, global_level, global_xp, city_id, active_job,
		        corporation_id, corporation_role, vip_until, daily_streak, last_daily_at,
		        last_active_at, created_at
		 FROM users WHERE telegram_user_id = $1`, telegramUserID).Scan(
		&u.ID, &u.TelegramUserID, &u.Balance, &u.GlobalLevel, &u.GlobalXP,
		&u.CityID, &u.ActiveJob, &u.CorporationID, &u.CorporationRole,
		&u.VipUntil, &u.DailyStreak, &u.LastDailyAt, &u.LastActiveAt, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UserService) GetSkills(ctx context.Context, userID int64) ([]models.UserSkill, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT user_id, direction, xp FROM user_skills WHERE user_id = $1 ORDER BY xp DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []models.UserSkill
	for rows.Next() {
		var sk models.UserSkill
		if err := rows.Scan(&sk.UserID, &sk.Direction, &sk.XP); err != nil {
			return nil, err
		}
		skills = append(skills, sk)
	}
	return skills, nil
}

func (s *UserService) AddXP(ctx context.Context, tx pgx.Tx, userID int64, direction string, xp int) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO user_skills (user_id, direction, xp) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, direction) DO UPDATE SET xp = user_skills.xp + $3`,
		userID, direction, xp)
	return err
}

// ClaimDailyBonus gives the daily login bonus with streak.
// Base 250₽, +50₽ per consecutive day up to 7 days (max 600₽).
func (s *UserService) ClaimDailyBonus(ctx context.Context, userID int64) (int, error) {
	var bonus int
	err := s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var streak int
		var lastDaily *time.Time
		err := tx.QueryRow(ctx,
			`SELECT daily_streak, last_daily_at FROM users WHERE id = $1 FOR UPDATE`,
			userID).Scan(&streak, &lastDaily)
		if err != nil {
			return err
		}

		now := time.Now()
		if lastDaily != nil {
			hoursSince := now.Sub(*lastDaily).Hours()
			if hoursSince < 20 {
				return fmt.Errorf("daily bonus already claimed recently")
			}
			if hoursSince < 48 {
				streak++
			} else {
				streak = 1
			}
		} else {
			streak = 1
		}
		if streak > 7 {
			streak = 7
		}

		bonus = 250 + (streak-1)*50
		if bonus > 600 {
			bonus = 600
		}

		// Credit via ledger
		if err := s.ledger.Credit(ctx, tx, "user", userID, bonus, "daily bonus"); err != nil {
			return err
		}

		_, err = tx.Exec(ctx,
			`UPDATE users SET daily_streak = $1, last_daily_at = NOW() WHERE id = $2`,
			streak, userID)
		return err
	})
	return bonus, err
}
