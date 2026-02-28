package user

import "github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"

type UserReadModel struct {
	ID       string `db:"id"        json:"id"`
	Email    string `db:"email"     json:"email"`
	PassHash string `db:"pass_hash" json:"-"` // never in API responses
	Active   bool   `db:"active"    json:"active"`
	database.BaseReadModel
}
