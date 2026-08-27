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

// Enroll pays for a program and starts it.
func (s *EducationService) Enroll(ctx context.Context, userID int64, programID string) error {
	return s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		// Check not already enrolled in active course
		var count int
		err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM user_education WHERE user_id = $1 AND program_id = $2 AND completed = FALSE`,
			userID, programID).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("уже enrolled в %s", programID)
		}

		// Get program
		var cost, requiredXP int
		err = tx.QueryRow(ctx,
			`SELECT cost, required_xp FROM education_programs WHERE id = $1`, programID).
			Scan(&cost, &requiredXP)
		if err != nil {
			return fmt.Errorf("program not found: %w", err)
		}

		// Check XP requirement (using first skill direction for the program)
		// For simplicity, we check the user's total XP in the relevant direction
		// The spec says "Требование XP препятствует мгновенной покупке высоких профессий"
		// We'll check total global XP as a simplified gate
		var globalXP int
		err = tx.QueryRow(ctx, `SELECT global_xp FROM users WHERE id = $1 FOR UPDATE`, userID).
			Scan(&globalXP)
		if err != nil {
			return err
		}
		if globalXP < requiredXP {
			return fmt.Errorf("недостаточно XP: %d/%d", globalXP, requiredXP)
		}

		// Debit cost
		if err := s.ledger.Debit(ctx, tx, "user", userID, cost,
			fmt.Sprintf("enroll: %s", programID)); err != nil {
			return err
		}

		// Create enrollment
		_, err = tx.Exec(ctx,
			`INSERT INTO user_education (user_id, program_id, progress, next_lesson_at)
			 VALUES ($1, $2, 0, NOW())`, userID, programID)
		return err
	})
}

// Study completes one lesson. Checks cooldown.
func (s *EducationService) Study(ctx context.Context, userID int64, programID string) (int, error) {
	var newProgress int
	err := s.pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var id int64
		var progress, lessonCount int
		var intervalHours int
		var nextLesson *time.Time

		err := tx.QueryRow(ctx,
			`SELECT ue.id, ue.progress, ep.lesson_count, ep.lesson_interval_hours, ue.next_lesson_at
			 FROM user_education ue JOIN education_programs ep ON ue.program_id = ep.id
			 WHERE ue.user_id = $1 AND ue.program_id = $2 AND ue.completed = FALSE
			 FOR UPDATE`, userID, programID).
			Scan(&id, &progress, &lessonCount, &intervalHours, &nextLesson)
		if err != nil {
			return fmt.Errorf("no active enrollment: %w", err)
		}

		// Check cooldown
		if nextLesson != nil && time.Now().Before(*nextLesson) {
			return fmt.Errorf("cooldown active, next lesson at %s", nextLesson.Format("15:04 02.01.2006"))
		}

		progress++
		newProgress = progress

		if progress >= lessonCount {
			// Course complete
			_, err = tx.Exec(ctx,
				`UPDATE user_education SET progress = $1, completed = TRUE, completed_at = NOW() WHERE id = $2`,
				progress, id)
			if err != nil {
				return err
			}
		} else {
			nextAt := time.Now().Add(time.Duration(intervalHours) * time.Hour)
			_, err = tx.Exec(ctx,
				`UPDATE user_education SET progress = $1, next_lesson_at = $2 WHERE id = $3`,
				progress, nextAt, id)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return newProgress, err
}

// GetUserEducation returns user's active and completed programs.
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
