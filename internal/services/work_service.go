package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"yachman/internal/models"
)

type WorkService struct {
	pool   *pgxpool.Pool
	ledger *LedgerService
	users  *UserService
}

func NewWorkService(pool *pgxpool.Pool, ledger *LedgerService, users *UserService) *WorkService {
	return &WorkService{pool: pool, ledger: ledger, users: users}
nfunc (s *WorkService) resolveInternalID(ctx context.Context, tx pgx.Tx, telegramID int64) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `SELECT id FROM users WHERE telegram_user_id = $1`, telegramID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("пользователь не найден: выполните /start")
	}
	return id, nil
}
}

func (s *WorkService) GetWorkDefinition(ctx context.Context, workID string) (*models.WorkDefinition, error) {
	var w models.WorkDefinition
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, direction, required_xp, duration_minutes, payout, xp_reward, resource_type, resource_amount
		 FROM work_definitions WHERE id = $1`, workID).Scan(
		&w.ID, &w.Name, &w.Direction, &w.RequiredXP, &w.DurationMinutes,
		&w.Payout, &w.XPReward, &w.ResourceType, &w.ResourceAmount)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *WorkService) ListWorksByDirection(ctx context.Context, direction string) ([]models.WorkDefinition, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, direction, required_xp, duration_minutes, payout, xp_reward, resource_type, resource_amount
		 FROM work_definitions WHERE direction = $1 ORDER BY required_xp`, direction)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var works []models.WorkDefinition
	for rows.Next() {
		var w models.WorkDefinition
		if err := rows.Scan(&w.ID, &w.Name, &w.Direction, &w.RequiredXP, &w.DurationMinutes,
			&w.Payout, &w.XPReward, &w.ResourceType, &w.ResourceAmount); err != nil {
			return nil, err
		}
		works = append(works, w)
	}
	return works, nil
}

func (s *WorkService) StartWork(ctx context.Context, telegramUserID int64, workID string, cityID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	userID, err := s.resolveInternalID(ctx, tx, telegramUserID)
	if err != nil {
		return err
	}

	var activeWork *string
	var corpID *int64
	err = tx.QueryRow(ctx,
		`SELECT active_job, corporation_id FROM users WHERE id = $1 FOR UPDATE`, userID).
		Scan(&activeWork, &corpID)
	if err != nil {
		return fmt.Errorf("пользователь не найден: %w", err)
	}
	if activeWork != nil {
		return fmt.Errorf("уже выполняется работа: %s", *activeWork)
	}
	if corpID != nil {
		return fmt.Errorf("владелец корпорации не может использовать /work")
	}

	var w models.WorkDefinition
	err = tx.QueryRow(ctx,
		`SELECT id, name, direction, required_xp, duration_minutes, payout, xp_reward, resource_type, resource_amount
		 FROM work_definitions WHERE id = $1`, workID).Scan(
		&w.ID, &w.Name, &w.Direction, &w.RequiredXP, &w.DurationMinutes,
		&w.Payout, &w.XPReward, &w.ResourceType, &w.ResourceAmount)
	if err != nil {
		return fmt.Errorf("работа не найдена: %w", err)
	}

	var dirXP int
	err = tx.QueryRow(ctx,
		`SELECT xp FROM user_skills WHERE user_id = $1 AND direction = $2`, userID, w.Direction).
		Scan(&dirXP)
	if err != nil {
		dirXP = 0
	}
	if dirXP < w.RequiredXP {
		return fmt.Errorf("недостаточно XP в направлении %s: %d/%d", w.Direction, dirXP, w.RequiredXP)
	}

	opID := uuid.New()
	startAt := time.Now()
	finishesAt := startAt.Add(time.Duration(w.DurationMinutes) * time.Minute)

	_, err = tx.Exec(ctx,
		`INSERT INTO work_runs (user_id, work_id, city_id, started_at, finishes_at, operation_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, workID, cityID, startAt, finishesAt, opID)
	if err != nil {
		return fmt.Errorf("создание записи работы: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE users SET active_job = $1 WHERE id = $2`, workID, userID)
	if err != nil {
		return fmt.Errorf("установка активной работы: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *WorkService) CompleteWork(ctx context.Context, runID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var userID int64
	var workID string
	var cityID int64
	var completed bool
	err = tx.QueryRow(ctx,
		`SELECT user_id, work_id, city_id, completed FROM work_runs WHERE id = $1 FOR UPDATE`, runID).
		Scan(&userID, &workID, &cityID, &completed)
	if err != nil {
		return fmt.Errorf("запись работы не найдена: %w", err)
	}
	if completed {
		return nil
	}

	var payout, xpReward, resAmount int
	var direction, resType string
	err = tx.QueryRow(ctx,
		`SELECT payout, xp_reward, direction, resource_type, resource_amount
		 FROM work_definitions WHERE id = $1`, workID).
		Scan(&payout, &xpReward, &direction, &resType, &resAmount)
	if err != nil {
		return fmt.Errorf("определение работы не найдено: %w", err)
	}

	if err := s.ledger.Credit(ctx, tx, "user", userID, payout,
		fmt.Sprintf("work payout: %s", workID)); err != nil {
		return fmt.Errorf("начисление оплаты: %w", err)
	}

	if err := s.users.AddXP(ctx, tx, userID, direction, xpReward); err != nil {
		return fmt.Errorf("добавление XP: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO city_resources (city_id, resource_id, stock) VALUES ($1, $2, $3)
		 ON CONFLICT (city_id, resource_id) DO UPDATE SET stock = city_resources.stock + $3`,
		cityID, resType, resAmount)
	if err != nil {
		return fmt.Errorf("добавление ресурса: %w", err)
	}

	var totalXP int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(xp), 0) FROM user_skills WHERE user_id = $1`, userID).
		Scan(&totalXP)
	if err != nil {
		totalXP = 0
	}
	level := calculateLevel(totalXP)
	_, err = tx.Exec(ctx,
		`UPDATE users SET global_xp = $1, global_level = $2, active_job = NULL WHERE id = $3`,
		totalXP, level, userID)
	if err != nil {
		return fmt.Errorf("обновление XP: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE work_runs SET completed = TRUE WHERE id = $1`, runID)
	if err != nil {
		return fmt.Errorf("завершение записи: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *WorkService) GetActiveWork(ctx context.Context, userID int64) (*models.WorkRun, string, error) {
	var run models.WorkRun
	var workName string
	err := s.pool.QueryRow(ctx,
		`SELECT wr.id, wr.user_id, wr.work_id, wr.city_id, wr.started_at, wr.finishes_at, wr.completed, wr.operation_id,
		        wd.name
		 FROM work_runs wr JOIN work_definitions wd ON wr.work_id = wd.id
		 WHERE wr.user_id = $1 AND wr.completed = FALSE
		 ORDER BY wr.started_at DESC LIMIT 1`, userID).
		Scan(&run.ID, &run.UserID, &run.WorkID, &run.CityID, &run.StartedAt,
			&run.FinishesAt, &run.Completed, &run.OperationID, &workName)
	if err != nil {
		return nil, "", err
	}
	return &run, workName, nil
}

func (s *WorkService) FinishExpiredWorks(ctx context.Context) (int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id FROM work_runs
		 WHERE completed = FALSE AND finishes_at <= NOW()`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}

	count := 0
	for _, id := range ids {
		if err := s.CompleteWork(ctx, id); err != nil {
			log.Printf("CompleteWork error for run %d: %v", id, err)
			continue
		}
		count++
	}
	return count, nil
}

// calculateLevel returns the player's level based on canonical XP thresholds.
// global_xp = floor(sum(all_skill_xp) / 10)
// Thresholds: L1=0, L2=25, L3=75, L4=150, L5=275, L6=450, L7=700, L8=1000, L9=1400, L10=1900
// After L10: +600 XP per level
func calculateLevel(totalXP int) int {
	thresholds := []int{0, 25, 75, 150, 275, 450, 700, 1000, 1400, 1900}
	for i := len(thresholds) - 1; i >= 0; i-- {
		if totalXP >= thresholds[i] {
			if i+1 < len(thresholds) {
				return i + 1
			}
			// Beyond L10: +600 XP per level
			remaining := totalXP - thresholds[i]
			return len(thresholds) + remaining/600
		}
	}
	return 1
}
