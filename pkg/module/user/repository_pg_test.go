package user_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers

func newMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return sqlx.NewDb(db, "postgres"), mock
}

var (
	now      = time.Now()
	mockCols = []string{"id", "email", "pass_hash", "active", "created_at", "updated_at"}
)

func mockRow(id int64, email string) *sqlmock.Rows {
	return sqlmock.NewRows(mockCols).AddRow(id, email, "hashed", true, now, now)
}

func mockRows(entries ...[]any) *sqlmock.Rows {
	rows := sqlmock.NewRows(mockCols)
	for _, e := range entries {
		row := make([]driver.Value, len(e))
		for i, v := range e {
			row[i] = v
		}
		rows.AddRow(row...)
	}
	return rows
}

// --- NewPgReadRepository ---

func TestNewPgReadRepository(t *testing.T) {
	db, _ := newMockDB(t)
	repo := user.NewPgReadRepository(db)
	assert.NotNil(t, repo)
}

// --- GetByID ---

func TestGetByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := user.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM users_read WHERE id = $1`)).
		WithArgs("user-1").
		WillReturnRows(mockRow(1, "alice@example.com"))

	got, err := repo.GetByID(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", got.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := user.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM users_read WHERE id = $1`)).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows(mockCols))

	got, err := repo.GetByID(context.Background(), "missing")
	assert.Nil(t, got)
	_ = err
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := user.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM users_read WHERE id = $1`)).
		WithArgs("user-1").
		WillReturnError(errors.New("db error"))

	got, err := repo.GetByID(context.Background(), "user-1")
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetByEmail ---

func TestGetByEmail_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := user.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM users_read WHERE email = $1`)).
		WithArgs("alice@example.com").
		WillReturnRows(mockRow(1, "alice@example.com"))

	got, err := repo.GetByEmail(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", got.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByEmail_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := user.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM users_read WHERE email = $1`)).
		WithArgs("nope@example.com").
		WillReturnRows(sqlmock.NewRows(mockCols))

	got, err := repo.GetByEmail(context.Background(), "nope@example.com")
	assert.Nil(t, got)
	_ = err
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByEmail_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := user.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM users_read WHERE email = $1`)).
		WithArgs("alice@example.com").
		WillReturnError(errors.New("db error"))

	got, err := repo.GetByEmail(context.Background(), "alice@example.com")
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- List ---

// listQueryMatcher matches any query containing our base SELECT.
var listQueryMatcher = sqlmock.QueryMatcherFunc(func(_, actualSQL string) error {
	matched, _ := regexp.MatchString(`SELECT \* FROM users_read`, actualSQL)
	if !matched {
		return errors.New("query does not match")
	}
	return nil
})

func newMockDBWithMatcher(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.NewWithDSN("sqlmock_db_list_"+t.Name(), sqlmock.QueryMatcherOption(listQueryMatcher))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return sqlx.NewDb(db, "postgres"), mock
}

func expectListQueries(mock sqlmock.Sqlmock, count int64, rows *sqlmock.Rows) {
	mock.ExpectQuery(`SELECT \* FROM users_read`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
	mock.ExpectQuery(`SELECT \* FROM users_read`).
		WillReturnRows(rows)
}

func TestList_Success(t *testing.T) {
	db, mock := newMockDBWithMatcher(t)
	repo := user.NewPgReadRepository(db)

	expectListQueries(mock, 2, mockRows(
		[]any{int64(1), "a@example.com", "hashed", true, now, now},
		[]any{int64(2), "b@example.com", "hashed", true, now, now},
	))

	req := user.ListRequest{Page: 1, PageSize: 10, SortBy: "created_at", SortDir: "asc"}
	resp, err := repo.List(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, int64(2), resp.Total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_DBError(t *testing.T) {
	db, mock := newMockDBWithMatcher(t)
	repo := user.NewPgReadRepository(db)

	mock.ExpectQuery(`SELECT \* FROM users_read`).
		WillReturnError(errors.New("db error"))

	req := user.ListRequest{Page: 1, PageSize: 10}
	resp, err := repo.List(context.Background(), req)
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_AllowedSortFields(t *testing.T) {
	sortFields := []string{"email", "created_at", "updated_at"}
	for _, sf := range sortFields {
		t.Run(sf, func(t *testing.T) {
			db, mock := newMockDBWithMatcher(t)
			repo := user.NewPgReadRepository(db)

			expectListQueries(mock, 0, sqlmock.NewRows(mockCols))

			req := user.ListRequest{Page: 1, PageSize: 5, SortBy: sf, SortDir: "desc"}
			resp, err := repo.List(context.Background(), req)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestList_DefaultSort(t *testing.T) {
	db, mock := newMockDBWithMatcher(t)
	repo := user.NewPgReadRepository(db)

	expectListQueries(mock, 0, sqlmock.NewRows(mockCols))

	req := user.ListRequest{} // zero value → Normalise falls back to "created_at"
	resp, err := repo.List(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}
