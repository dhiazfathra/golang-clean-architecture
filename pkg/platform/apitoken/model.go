package apitoken

import (
	"time"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
)

type APIToken struct {
	ID          int64     `db:"id"           json:"id"`
	Name        string    `db:"name"         json:"name"`
	TokenHash   string    `db:"token_hash"   json:"-"`
	TokenPrefix string    `db:"token_prefix" json:"token_prefix"`
	UserID      string    `db:"user_id"      json:"user_id"`
	ExpiresAt   time.Time `db:"expires_at"   json:"expires_at"`
	database.BaseReadModel
}
