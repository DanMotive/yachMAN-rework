package models

import "time"

// BusinessType defines a seed-loaded business template.
type BusinessType struct {
	TypeID      string `db:"type_id"`
	Name        string `db:"name"`
	InputARes   string `db:"input_a_resource"`
	InputAAmount int   `db:"input_a_amount"`
	InputBRes   string `db:"input_b_resource"`
	InputBAmount int   `db:"input_b_amount"`
	OutputRes   string `db:"output_resource"`
	OutputAmount int   `db:"output_amount"`
	NPCStaff    int    `db:"npc_staff"`
}

type Business struct {
	ID            int64      `db:"id"`
	CityID        int64      `db:"city_id"`
	TypeID        string     `db:"type_id"`
	Name          string     `db:"name"`
	OwnerUserID   *int64     `db:"owner_user_id"`
	CorporationID *int64     `db:"corporation_id"`
	PowerPct      int        `db:"power_pct"`
	LastTickAt    *time.Time `db:"last_tick_at"`
	CreatedAt     time.Time  `db:"created_at"`
}

type BusinessStaff struct {
	ID         int64     `db:"id"`
	BusinessID int64     `db:"business_id"`
	UserID     int64     `db:"user_id"`
	Position   string    `db:"position"`
	Salary     int       `db:"salary"`
	HiredAt    time.Time `db:"hired_at"`
}
