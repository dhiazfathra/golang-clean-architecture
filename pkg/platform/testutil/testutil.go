package testutil

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/valkey-io/valkey-go"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/config"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/observability"
)

func init() {
	observability.InitNoop()
}

func SetupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := config.GetOr("TEST_DATABASE_URL",
		"postgres://app:app@localhost:5432/app_test?sslmode=disable")
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("testutil: close db: %v", err)
		}
	})
	return db
}

func SetupTestValkey(t *testing.T) valkey.Client {
	t.Helper()
	addr := config.GetOr("TEST_VALKEY_URL", "localhost:6379")
	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{addr}})
	if err != nil {
		t.Skipf("valkey not available: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}
