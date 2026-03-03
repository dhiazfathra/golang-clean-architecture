package order_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/order"
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
	mockCols = []string{"id", "user_id", "status", "total", "created_at", "updated_at"}
)

func mockRow(id int64, status string, total float64) *sqlmock.Rows {
	return sqlmock.NewRows(mockCols).AddRow(id, int64(99), status, total, now, now)
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
	repo := order.NewPgReadRepository(db)
	assert.NotNil(t, repo)
}

// --- GetByID ---

func TestGetByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := order.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM orders_read WHERE id = $1`)).
		WithArgs(int64(1)).
		WillReturnRows(mockRow(1, "pending", 99.99))

	got, err := repo.GetByID(context.Background(), "1")
	require.NoError(t, err)
	assert.Equal(t, "pending", got.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := order.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM orders_read WHERE id = $1`)).
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows(mockCols))

	got, err := repo.GetByID(context.Background(), "999")
	assert.Nil(t, got)
	_ = err
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := order.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM orders_read WHERE id = $1`)).
		WithArgs(int64(1)).
		WillReturnError(errors.New("db error"))

	got, err := repo.GetByID(context.Background(), "1")
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_InvalidID(t *testing.T) {
	db, _ := newMockDB(t)
	repo := order.NewPgReadRepository(db)

	got, err := repo.GetByID(context.Background(), "not-a-number")
	assert.Nil(t, got)
	assert.Error(t, err)
}

// --- List ---

var listQueryMatcher = sqlmock.QueryMatcherFunc(func(_, actualSQL string) error {
	matched, _ := regexp.MatchString(`SELECT \* FROM orders_read`, actualSQL)
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
	mock.ExpectQuery(`SELECT \* FROM orders_read`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
	mock.ExpectQuery(`SELECT \* FROM orders_read`).
		WillReturnRows(rows)
}

func TestList_Success(t *testing.T) {
	db, mock := newMockDBWithMatcher(t)
	repo := order.NewPgReadRepository(db)

	expectListQueries(mock, 2, mockRows(
		[]any{int64(1), int64(99), "pending", 50.00, now, now},
		[]any{int64(2), int64(99), "completed", 120.00, now, now},
	))

	req := order.ListRequest{Page: 1, PageSize: 10, SortBy: "created_at", SortDir: "asc"}
	resp, err := repo.List(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, int64(2), resp.Total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_DBError(t *testing.T) {
	db, mock := newMockDBWithMatcher(t)
	repo := order.NewPgReadRepository(db)

	mock.ExpectQuery(`SELECT \* FROM orders_read`).
		WillReturnError(errors.New("db error"))

	req := order.ListRequest{Page: 1, PageSize: 10}
	resp, err := repo.List(context.Background(), req)
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_AllowedSortFields(t *testing.T) {
	sortFields := []string{"total", "status", "created_at"}
	for _, sf := range sortFields {
		t.Run(sf, func(t *testing.T) {
			db, mock := newMockDBWithMatcher(t)
			repo := order.NewPgReadRepository(db)

			expectListQueries(mock, 0, sqlmock.NewRows(mockCols))

			req := order.ListRequest{Page: 1, PageSize: 5, SortBy: sf, SortDir: "desc"}
			resp, err := repo.List(context.Background(), req)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestList_DefaultSort(t *testing.T) {
	db, mock := newMockDBWithMatcher(t)
	repo := order.NewPgReadRepository(db)

	expectListQueries(mock, 0, sqlmock.NewRows(mockCols))

	req := order.ListRequest{} // zero value → Normalise falls back to "created_at"
	resp, err := repo.List(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}
