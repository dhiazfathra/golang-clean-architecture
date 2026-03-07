package testutil

import (
	"errors"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// NewMockDB creates a sqlmock-backed *sqlx.DB using the "sqlmock" driver name.
// This is suitable for generic unit tests that only need to verify query structure,
// argument binding, and row scanning — without any PostgreSQL-specific behavior.
//
// Because sqlx treats the "sqlmock" driver as an unknown dialect, it uses the
// QUESTION bind type (?) for placeholder rebinding. Use this when:
//   - Your queries use raw SQL with no placeholder rebinding via sqlx.Rebind
//   - You want a lightweight mock that is not coupled to any real database dialect
//   - You are testing non-Postgres code paths (e.g. generic repositories)
//
// Example:
//
//	func TestGetUser(t *testing.T) {
//	    db, mock := testutil.NewMockDB(t)
//
//	    rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Alice")
//	    mock.ExpectQuery(`SELECT \* FROM users WHERE id = ?`).
//	        WithArgs(1).
//	        WillReturnRows(rows)
//
//	    user, err := repo.GetUser(db, 1)
//	    require.NoError(t, err)
//	    require.Equal(t, "Alice", user.Name)
//	}
func NewMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rawDB.Close() })
	return sqlx.NewDb(rawDB, "sqlmock"), mock
}

// NewMockPostgresDB creates a sqlmock-backed *sqlx.DB using the "postgres" driver name.
// This is the correct choice when testing code that targets PostgreSQL specifically,
// because sqlx will apply the DOLLAR bind type ($1, $2, ...) to match Postgres
// placeholder syntax. Use this when:
//   - Your queries or repository layer use $1-style placeholders
//   - You call sqlx.Rebind or rely on sqlx dialect-aware helpers (e.g. In, NamedQuery)
//   - You are testing pg-specific repositories, projectors, or query builders
//
// Example:
//
//	func TestCreateOrder(t *testing.T) {
//	    db, mock := testutil.NewMockPostgresDB(t)
//
//	    mock.ExpectExec(`INSERT INTO orders \(user_id, total\) VALUES \(\$1, \$2\)`).
//	        WithArgs(42, 99.99).
//	        WillReturnResult(sqlmock.NewResult(1, 1))
//
//	    err := repo.CreateOrder(db, 42, 99.99)
//	    require.NoError(t, err)
//	}
func NewMockPostgresDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rawDB.Close() })
	return sqlx.NewDb(rawDB, "postgres"), mock
}

// listQueryMatcher returns a sqlmock.QueryMatcherFunc that matches any query containing
// a SELECT * FROM the given table name.
func listQueryMatcher(table string) sqlmock.QueryMatcherFunc {
	return sqlmock.QueryMatcherFunc(func(_, actualSQL string) error {
		matched, _ := regexp.MatchString(`SELECT \* FROM `+regexp.QuoteMeta(table), actualSQL)
		if !matched {
			return errors.New("query does not match")
		}
		return nil
	})
}

// NewMockDBWithMatcher creates a sqlmock database with a custom query matcher for use in
// unit tests that exercise code paths involving *sqlx.DB (projectors, pg repositories).
// The underlying *sql.DB is closed automatically via t.Cleanup.
func NewMockDBWithMatcher(t *testing.T, table string) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.NewWithDSN("sqlmock_db_list_"+t.Name(), sqlmock.QueryMatcherOption(listQueryMatcher(table)))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return sqlx.NewDb(db, "postgres"), mock
}
