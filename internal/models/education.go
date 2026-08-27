package models

type EducationProgram struct {
	ID                   string `db:"id"`
	Name                 string `db:"name"`
	Direction            string `db:"direction"`
	RequiredXP           int    `db:"required_xp"`
	Cost                 int    `db:"cost"`
	LessonCount          int    `db:"lesson_count"`
	LessonIntervalHours  int    `db:"lesson_interval_hours"`
}
