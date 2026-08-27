package models

import "time"

type Corporation struct {
	ID                  int64     `db:"id"`
	Name                string    `db:"name"`
	OwnerUserID         int64     `db:"owner_user_id"`
	Balance             int       `db:"balance"`
	RegistrationFeePaid int       `db:"registration_fee_paid"`
	TotalShares         int       `db:"total_shares"`
	CreatedAt           time.Time `db:"created_at"`
}

type CorporationStaff struct {
	ID              int64     `db:"id"`
	CorporationID   int64     `db:"corporation_id"`
	UserID          int64     `db:"user_id"`
	Position        string    `db:"position"`
	Salary          int       `db:"salary"`
	HiredAt         time.Time `db:"hired_at"`
}
