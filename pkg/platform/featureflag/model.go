package featureflag

import (
	"encoding/json"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
)

type Flag struct {
	ID          int64           `db:"id" json:"id"`
	Key         string          `db:"key" json:"key"`
	Enabled     bool            `db:"enabled" json:"enabled"`
	Description string          `db:"description" json:"description"`
	Metadata    json.RawMessage `db:"metadata" json:"metadata"`
	database.BaseReadModel
}
