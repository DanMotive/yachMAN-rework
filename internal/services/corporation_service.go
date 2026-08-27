package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"yachman/internal/models"
)

type CorporationService struct {
	pool   *pgxpool.Pool
	ledger *LedgerService
}

func NewCorporationService(pool *pgxpool.Pool, ledger *LedgerService) *CorporationService {
	return &CorporationService{pool: pool, ledger: ledger}
}

func (s *CorporationService) CreateCorporation(ctx context.Context, ownerUserID int64, name string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var existingCorp *int64
	_ = tx.QueryRow(ctx,
		`SELECT id FROM corporations WHERE owner_user_id = $1`, ownerUserID).Scan(&existingCorp)
	if existingCorp != nil {
		return fmt.Errorf("вы уже владеете корпорацией")
	}

	if err := s.ledger.Debit(ctx, tx, "user", ownerUserID, 6000000,
		"corporation registration"); err != nil {
		return err
	}

	var corpID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO corporations (name, owner_user_id, balance, registration_fee_paid, total_shares)
		 VALUES ($1, $2, 5000000, 1000000, 10000000) RETURNING id`,
		name, ownerUserID).Scan(&corpID)
	if err != nil {
		return err
	}

	if err := s.ledger.Credit(ctx, tx, "corporation", corpID, 5000000,
		"charter capital from founder"); err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO shares (corporation_id, user_id, amount) VALUES ($1, $2, 10000000)`,
		corpID, ownerUserID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE users SET corporation_id = $1, corporation_role = 'owner' WHERE id = $2`,
		corpID, ownerUserID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *CorporationService) GetCorporation(ctx context.Context, corpID int64) (*models.Corporation, error) {
	var c models.Corporation
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, owner_user_id, balance, registration_fee_paid, total_shares, created_at
		 FROM corporations WHERE id = $1`, corpID).Scan(
		&c.ID, &c.Name, &c.OwnerUserID, &c.Balance, &c.RegistrationFeePaid,
		&c.TotalShares, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CorporationService) HireEmployee(ctx context.Context, corpID, userID int64, position string, salary int) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO corporation_staff (corporation_id, user_id, position, salary)
		 VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`, corpID, userID, position, salary)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`UPDATE users SET corporation_id = $1, corporation_role = 'employee' WHERE id = $2 AND corporation_id IS NULL`,
		corpID, userID)
	return err
}

func (s *CorporationService) FireEmployee(ctx context.Context, corpID, userID int64) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM corporation_staff WHERE corporation_id = $1 AND user_id = $2`, corpID, userID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`UPDATE users SET corporation_id = NULL, corporation_role = NULL WHERE id = $1 AND corporation_id = $2`,
		userID, corpID)
	return err
}

func (s *CorporationService) GetStaff(ctx context.Context, corpID int64) ([]models.CorporationStaff, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, corporation_id, user_id, position, salary, hired_at
		 FROM corporation_staff WHERE corporation_id = $1 ORDER BY hired_at`, corpID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var staff []models.CorporationStaff
	for rows.Next() {
		var m models.CorporationStaff
		if err := rows.Scan(&m.ID, &m.CorporationID, &m.UserID, &m.Position, &m.Salary, &m.HiredAt); err != nil {
			return nil, err
		}
		staff = append(staff, m)
	}
	return staff, nil
}

func (s *CorporationService) PaySalaries(ctx context.Context) error {
	rows, err := s.pool.Query(ctx,
		`SELECT cs.id, cs.user_id, cs.salary, cs.corporation_id
		 FROM corporation_staff cs JOIN corporations c ON cs.corporation_id = c.id
		 WHERE cs.salary > 0`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var staffID, userID int64
		var salary int
		var corpID int64
		if err := rows.Scan(&staffID, &userID, &salary, &corpID); err != nil {
			continue
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			continue
		}
		err = s.ledger.Transfer(ctx, tx, "corporation", corpID, salary, "user", userID, salary,
			fmt.Sprintf("salary: staff %d", staffID))
		if err != nil {
			tx.Rollback(ctx)
			continue
		}
		tx.Commit(ctx)
	}
	return nil
}
