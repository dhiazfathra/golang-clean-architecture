package envvar

import (
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
)

type EnvVar struct {
	ID       int64  `db:"id" json:"id"`
	Platform string `db:"platform" json:"platform"`
	Key      string `db:"key" json:"key"`
	Value    string `db:"value" json:"value"`
	database.BaseReadModel
}
