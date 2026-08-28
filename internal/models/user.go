package models

import "time"

type User struct {
	ID              int64      `db:"id" json:"id"`
	TelegramUserID  int64      `db:"telegram_user_id" json:"telegram_user_id"`
	Balance         int        `db:"balance" json:"balance"`
	GlobalLevel     int        `db:"global_level" json:"global_level"`
	GlobalXP        int        `db:"global_xp" json:"global_xp"`
	CityID          *int64     `db:"city_id" json:"city_id"`
	ActiveJob       *string    `db:"active_job" json:"active_job"`
	CorporationID   *int64     `db:"corporation_id" json:"corporation_id"`
	CorporationRole *string    `db:"corporation_role" json:"corporation_role"`
	VipUntil        *time.Time `db:"vip_until" json:"vip_until"`
	DailyStreak     int        `db:"daily_streak" json:"daily_streak"`
	LastDailyAt     *time.Time `db:"last_daily_at" json:"last_daily_at"`
	LastActiveAt    *time.Time `db:"last_active_at" json:"last_active_at"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

type UserSkill struct {
	UserID    int64  `db:"user_id" json:"user_id"`
	Direction string `db:"direction" json:"direction"`
	XP        int    `db:"xp" json:"xp"`
}

type UserEducation struct {
	ID           int64      `db:"id" json:"id"`
	UserID       int64      `db:"user_id" json:"user_id"`
	ProgramID    string     `db:"program_id" json:"program_id"`
	Progress     int        `db:"progress" json:"progress"`
	Completed    bool       `db:"completed" json:"completed"`
	NextLessonAt *time.Time `db:"next_lesson_at" json:"next_lesson_at"`
	StartedAt    time.Time  `db:"started_at" json:"started_at"`
	CompletedAt  *time.Time `db:"completed_at" json:"completed_at"`
}
