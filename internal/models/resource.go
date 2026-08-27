package models

type Resource struct {
	ID         string `db:"id"`
	Name       string `db:"name"`
	BasePrice  int    `db:"base_price"`
}

type CityResource struct {
	CityID     int64  `db:"city_id"`
	ResourceID string `db:"resource_id"`
	Stock      int    `db:"stock"`
	Demand     int    `db:"demand"`
	LastPrice  int    `db:"last_price"`
}
