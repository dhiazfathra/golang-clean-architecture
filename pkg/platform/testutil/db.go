package testutil

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// NewMockDB creates a regexp-matching sqlmock database and an sqlx wrapper for use in
// unit tests that exercise code paths involving *sqlx.DB (projectors, pg repositories).
// The underlying *sql.DB is closed automatically via t.Cleanup.
func NewMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rawDB.Close() })
	return sqlx.NewDb(rawDB, "sqlmock"), mock
}
