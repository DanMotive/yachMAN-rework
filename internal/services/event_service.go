package services

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"yachman/internal/models"
)

type EventService struct {
	pool *pgxpool.Pool
}

func NewEventService(pool *pgxpool.Pool) *EventService {
	return &EventService{pool: pool}
}

func (s *EventService) GetActiveEvents(ctx context.Context) ([]models.Event, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, type, name, description, city_id, start_at, end_at, created_at
		 FROM events WHERE start_at <= NOW() AND end_at >= NOW() ORDER BY start_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(&e.ID, &e.Type, &e.Name, &e.Description, &e.CityID,
			&e.StartAt, &e.EndAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (s *EventService) GetRecentWorldEvents(ctx context.Context, limit int) ([]models.Event, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, type, name, description, city_id, start_at, end_at, created_at
		 FROM events WHERE city_id IS NULL ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(&e.ID, &e.Type, &e.Name, &e.Description, &e.CityID,
			&e.StartAt, &e.EndAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
