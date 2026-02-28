package order

import "github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"

type OrderReadModel struct {
	ID     int64   `db:"id"      json:"id"`
	UserID int64   `db:"user_id" json:"user_id"`
	Status string  `db:"status"  json:"status"`
	Total  float64 `db:"total"   json:"total"`
	database.BaseReadModel
}
