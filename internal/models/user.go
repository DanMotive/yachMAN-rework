package models

import "time"

type User struct {
	ID              int64      `db:"id"`
	TelegramUserID  int64      `db:"telegram_user_id"`
	Balance         int        `db:"balance"`
	GlobalLevel     int        `db:"global_level"`
	GlobalXP        int        `db:"global_xp"`
	CityID          *int64     `db:"city_id"`
	ActiveJob       *string    `db:"active_job"`
	CorporationID   *int64     `db:"corporation_id"`
	CorporationRole *string    `db:"corporation_role"`
	VipUntil        *time.Time `db:"vip_until"`
	DailyStreak     int        `db:"daily_streak"`
	LastDailyAt     *time.Time `db:"last_daily_at"`
	LastActiveAt    *time.Time `db:"last_active_at"`
	CreatedAt       time.Time  `db:"created_at"`
}

type UserSkill struct {
	UserID    int64  `db:"user_id"`
	Direction string `db:"direction"`
	XP        int    `db:"xp"`
}

type UserEducation struct {
	ID           int64      `db:"id"`
	UserID       int64      `db:"user_id"`
	ProgramID    string     `db:"program_id"`
	Progress     int        `db:"progress"`
	Completed    bool       `db:"completed"`
	NextLessonAt *time.Time `db:"next_lesson_at"`
	StartedAt    time.Time  `db:"started_at"`
	CompletedAt  *time.Time `db:"completed_at"`
}
