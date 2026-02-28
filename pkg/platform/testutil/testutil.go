package testutil

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/observability"
)

func init() {
	observability.InitNoop()
}

func SetupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := "postgres://app:app@localhost:5432/app_test?sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func SetupTestValkey(t *testing.T) string {
	t.Helper()
	return "localhost:6379"
}
