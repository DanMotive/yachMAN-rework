package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"yachman/internal/models"
)

type EducationService struct {
	pool   *pgxpool.Pool
	ledger *LedgerService
}

func NewEducationService(pool *pgxpool.Pool, ledger *LedgerService) *EducationService {
	return &EducationService{pool: pool, ledger: ledger}
nfunc (s *EducationService) resolveInternalID(ctx context.Context, tx pgx.Tx, telegramID int64) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `SELECT id FROM users WHERE telegram_user_id = $1`, telegramID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("пользователь не найден")
	}
	return id, nil
}
}

func (s *EducationService) GetProgram(ctx context.Context, programID string) (*models.EducationProgram, error) {
	var p models.EducationProgram
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, direction, required_xp, cost, lesson_count, lesson_interval_hours
		 FROM education_programs WHERE id = $1`, programID).Scan(
		&p.ID, &p.Name, &p.Direction, &p.RequiredXP, &p.Cost, &p.LessonCount, &p.LessonIntervalHours)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *EducationService) ListPrograms(ctx context.Context) ([]models.EducationProgram, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, direction, required_xp, cost, lesson_count, lesson_interval_hours
		 FROM education_programs ORDER BY cost`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var programs []models.EducationProgram
	for rows.Next() {
		var p models.EducationProgram
		if err := rows.Scan(&p.ID, &p.Name, &p.Direction, &p.RequiredXP, &p.Cost, &p.LessonCount, &p.LessonIntervalHours); err != nil {
			return nil, err
		}
		programs = append(programs, p)
	}
	return programs, nil
}

func (s *EducationService) Enroll(ctx context.Context, telegramUserID int64, programID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	userID, err := s.resolveInternalID(ctx, tx, telegramUserID)
	if err != nil {
		return err
	}

	var count int
	err = tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_education WHERE user_id = $1 AND program_id = $2 AND completed = FALSE`,
		userID, programID).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("уже записан на %s", programID)
	}

	var cost, requiredXP int
	err = tx.QueryRow(ctx,
		`SELECT cost, required_xp FROM education_programs WHERE id = $1`, programID).
		Scan(&cost, &requiredXP)
	if err != nil {
		return fmt.Errorf("программа не найдена: %w", err)
	}

	var globalXP int
	err = tx.QueryRow(ctx, `SELECT global_xp FROM users WHERE id = $1 FOR UPDATE`, userID).
		Scan(&globalXP)
	if err != nil {
		return err
	}
	if globalXP < requiredXP {
		return fmt.Errorf("недостаточно XP: %d/%d", globalXP, requiredXP)
	}

	if err := s.ledger.Debit(ctx, tx, "user", userID, cost,
		fmt.Sprintf("enroll: %s", programID)); err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO user_education (user_id, program_id, progress, next_lesson_at)
		 VALUES ($1, $2, 0, NOW())`, userID, programID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *EducationService) Study(ctx context.Context, telegramUserID int64, programID string) (int, error) {
	var newProgress int

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	userID, err := s.resolveInternalID(ctx, tx, telegramUserID)
	if err != nil {
		return 0, err
	}

	var id int64
	var progress, lessonCount, intervalHours int
	var nextLesson *time.Time

	err = tx.QueryRow(ctx,
		`SELECT ue.id, ue.progress, ep.lesson_count, ep.lesson_interval_hours, ue.next_lesson_at
		 FROM user_education ue JOIN education_programs ep ON ue.program_id = ep.id
		 WHERE ue.user_id = $1 AND ue.program_id = $2 AND ue.completed = FALSE
		 FOR UPDATE`, userID, programID).
		Scan(&id, &progress, &lessonCount, &intervalHours, &nextLesson)
	if err != nil {
		return 0, fmt.Errorf("нет активной записи: %w", err)
	}

	if nextLesson != nil && time.Now().Before(*nextLesson) {
		return 0, fmt.Errorf("кулдаун, следующий урок в %s", nextLesson.Format("15:04 02.01.2006"))
	}

	progress++
	newProgress = progress

	if progress >= lessonCount {
		_, err = tx.Exec(ctx,
			`UPDATE user_education SET progress = $1, completed = TRUE, completed_at = NOW() WHERE id = $2`,
			progress, id)
	} else {
		nextAt := time.Now().Add(time.Duration(intervalHours) * time.Hour)
		_, err = tx.Exec(ctx,
			`UPDATE user_education SET progress = $1, next_lesson_at = $2 WHERE id = $3`,
			progress, nextAt, id)
	}
	if err != nil {
		return 0, err
	}

	return newProgress, tx.Commit(ctx)
}

func (s *EducationService) GetUserEducation(ctx context.Context, userID int64) ([]models.UserEducation, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, program_id, progress, completed, next_lesson_at, started_at, completed_at
		 FROM user_education WHERE user_id = $1 ORDER BY completed, started_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var educations []models.UserEducation
	for rows.Next() {
		var e models.UserEducation
		if err := rows.Scan(&e.ID, &e.UserID, &e.ProgramID, &e.Progress, &e.Completed,
			&e.NextLessonAt, &e.StartedAt, &e.CompletedAt); err != nil {
			return nil, err
		}
		educations = append(educations, e)
	}
	return educations, nil
}
