package models

import "time"

type City struct {
	ID                int64      `db:"id"`
	ChatID            int64      `db:"chat_id"`
	Name              string     `db:"name"`
	Description       string     `db:"description"`
	Level             string     `db:"level"`
	NPCPopulation     int        `db:"npc_population"`
	DevelopmentPoints int        `db:"development_points"`
	Treasury          int        `db:"treasury"`
	TaxRateBusiness   float64    `db:"tax_rate_business"`
	TaxRateCorporate  float64    `db:"tax_rate_corporate"`
	TaxRateIncome     float64    `db:"tax_rate_income"`
	LastTaxChangeAt   *time.Time `db:"last_tax_change_at"`
	AccessMode        string     `db:"access_mode"`
	MayorUserID       *int64     `db:"mayor_user_id"`
	PublicListing     bool       `db:"public_listing"`
	CreatedAt         time.Time  `db:"created_at"`
}

type CityMember struct {
	CityID   int64     `db:"city_id"`
	UserID   int64     `db:"user_id"`
	Role     string    `db:"role"`
	JoinedAt time.Time `db:"joined_at"`
}

type CityAdmin struct {
	CityID     int64     `db:"city_id"`
	UserID     int64     `db:"user_id"`
	Position   string    `db:"position"`
	AppointedAt time.Time `db:"appointed_at"`
}

type CityProject struct {
	ID          int64     `db:"id"`
	CityID      int64     `db:"city_id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Cost        int       `db:"cost"`
	Progress    int       `db:"progress"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
}

type CityTaxLog struct {
	ID          int64     `db:"id"`
	CityID      int64     `db:"city_id"`
	TaxType     string    `db:"tax_type"`
	OldRate     float64   `db:"old_rate"`
	NewRate     float64   `db:"new_rate"`
	ChangedBy   int64     `db:"changed_by"`
	ChangedAt   time.Time `db:"changed_at"`
}
