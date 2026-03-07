package database

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/observability"
)

func init() {
	observability.InitNoop()
}

func newMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rawDB.Close() })
	return sqlx.NewDb(rawDB, "sqlmock"), mock
}

// --- BaseReadModel ---

func TestBaseReadModelFields(t *testing.T) {
	m := BaseReadModel{
		CreatedAt: time.Now(),
		CreatedBy: "alice",
		UpdatedAt: time.Now(),
		UpdatedBy: "bob",
		IsDeleted: false,
	}
	assert.Equal(t, "alice", m.CreatedBy)
	assert.Equal(t, "bob", m.UpdatedBy)
	assert.False(t, m.IsDeleted)
}

// --- AuditMeta ---

func TestAuditMeta(t *testing.T) {
	now := time.Now()
	am := AuditMeta{At: now, By: "user1"}
	assert.Equal(t, now, am.At)
	assert.Equal(t, "user1", am.By)
}

// --- UpsertReadModel ---

func TestUpsertReadModel(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))

	err := UpsertReadModel(context.Background(), db,
		"INSERT INTO users (id, name) VALUES (:id, :name)",
		map[string]any{"id": 1, "name": "Alice"})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertReadModelError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec("INSERT INTO users").WillReturnError(errors.New("upsert fail"))

	err := UpsertReadModel(context.Background(), db,
		"INSERT INTO users (id, name) VALUES (:id, :name)",
		map[string]any{"id": 1, "name": "Alice"})
	assert.Error(t, err)
}

// --- CRUD functions ---

type testRow struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	IsDeleted bool   `db:"is_deleted"`
}

func TestGet(t *testing.T) {
	db, mock := newMockDB(t)

	rows := sqlmock.NewRows([]string{"id", "name", "is_deleted"}).
		AddRow(1, "Alice", false)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := Get[testRow](context.Background(), db, "SELECT id, name, is_deleted FROM users WHERE id = $1", 1)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Alice", result.Name)
}

func TestGetError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("get fail"))

	_, err := Get[testRow](context.Background(), db, "SELECT id, name, is_deleted FROM users WHERE id = $1", 1)
	assert.Error(t, err)
}

func TestSelect(t *testing.T) {
	db, mock := newMockDB(t)

	rows := sqlmock.NewRows([]string{"id", "name", "is_deleted"}).
		AddRow(1, "Alice", false).
		AddRow(2, "Bob", false)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	results, err := Select[testRow](context.Background(), db, "SELECT id, name, is_deleted FROM users")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestSelectError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("select fail"))

	_, err := Select[testRow](context.Background(), db, "SELECT id, name, is_deleted FROM users")
	assert.Error(t, err)
}

func TestGetIncludingDeleted(t *testing.T) {
	db, mock := newMockDB(t)

	rows := sqlmock.NewRows([]string{"id", "name", "is_deleted"}).
		AddRow(1, "Alice", true)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := GetIncludingDeleted[testRow](context.Background(), db, "SELECT id, name, is_deleted FROM users WHERE id = $1", 1)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsDeleted)
}

func TestGetIncludingDeletedError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("fail"))

	_, err := GetIncludingDeleted[testRow](context.Background(), db, "SELECT id, name, is_deleted FROM users WHERE id = $1", 1)
	assert.Error(t, err)
}

func TestSelectIncludingDeleted(t *testing.T) {
	db, mock := newMockDB(t)

	rows := sqlmock.NewRows([]string{"id", "name", "is_deleted"}).
		AddRow(1, "Alice", true)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	results, err := SelectIncludingDeleted[testRow](context.Background(), db, "SELECT id, name, is_deleted FROM users")
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSelectIncludingDeletedError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("fail"))

	_, err := SelectIncludingDeleted[testRow](context.Background(), db, "SELECT id, name, is_deleted FROM users")
	assert.Error(t, err)
}

func TestExec(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec("DELETE FROM users").WillReturnResult(sqlmock.NewResult(0, 1))

	err := Exec(context.Background(), db, "DELETE FROM users WHERE id = $1", 1)
	require.NoError(t, err)
}

func TestExecError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec("DELETE FROM users").WillReturnError(errors.New("exec fail"))

	err := Exec(context.Background(), db, "DELETE FROM users WHERE id = $1", 1)
	assert.Error(t, err)
}

func TestNamedExec(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec("UPDATE users").WillReturnResult(sqlmock.NewResult(0, 1))

	err := NamedExec(context.Background(), db, "UPDATE users SET name = :name WHERE id = :id",
		map[string]any{"id": 1, "name": "Bob"})
	require.NoError(t, err)
}

func TestNamedExecError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec("UPDATE users").WillReturnError(errors.New("named fail"))

	err := NamedExec(context.Background(), db, "UPDATE users SET name = :name WHERE id = :id",
		map[string]any{"id": 1, "name": "Bob"})
	assert.Error(t, err)
}

// --- Pagination ---

func TestPageRequestNormalise(t *testing.T) {
	tests := []struct {
		name        string
		input       PageRequest
		wantPage    int
		wantSize    int
		wantSortDir string
		wantSortBy  string
	}{
		{"defaults", PageRequest{}, 1, 20, "asc", "created_at"},
		{"negative page", PageRequest{Page: -5, PageSize: 10}, 1, 10, "asc", "created_at"},
		{"oversized page_size", PageRequest{Page: 2, PageSize: 999}, 2, 20, "asc", "created_at"},
		{"desc preserved", PageRequest{Page: 1, PageSize: 10, SortDir: "desc", SortBy: "name"}, 1, 10, "desc", "name"},
		{"invalid dir", PageRequest{Page: 1, PageSize: 10, SortDir: "xyz"}, 1, 10, "asc", "created_at"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.input
			req.Normalise("created_at")
			assert.Equal(t, tc.wantPage, req.Page)
			assert.Equal(t, tc.wantSize, req.PageSize)
			assert.Equal(t, tc.wantSortDir, req.SortDir)
			assert.Equal(t, tc.wantSortBy, req.SortBy)
		})
	}
}

func TestPaginatedSelect(t *testing.T) {
	db, mock := newMockDB(t)

	// Count query
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Data query
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "is_deleted"}).
			AddRow(1, "Alice", false).
			AddRow(2, "Bob", false))

	req := PageRequest{Page: 1, PageSize: 10, SortBy: "name", SortDir: "asc"}
	allowed := map[string]string{"name": "name", "created_at": "created_at"}

	result, err := PaginatedSelect[testRow](context.Background(), db,
		"SELECT id, name, is_deleted FROM users", req, allowed)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, 1, result.TotalPages)
}

func TestPaginatedSelectUnknownSort(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "is_deleted"}))

	req := PageRequest{Page: 1, PageSize: 10, SortBy: "unknown_col", SortDir: "asc"}
	allowed := map[string]string{"name": "name"}

	result, err := PaginatedSelect[testRow](context.Background(), db,
		"SELECT id, name, is_deleted FROM users", req, allowed)
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
}

func TestPaginatedSelectCountError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("count fail"))

	req := PageRequest{Page: 1, PageSize: 10, SortBy: "name", SortDir: "asc"}
	allowed := map[string]string{"name": "name"}

	_, err := PaginatedSelect[testRow](context.Background(), db,
		"SELECT id, name, is_deleted FROM users", req, allowed)
	assert.Error(t, err)
}

func TestPaginatedSelectDataError(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT").WillReturnError(errors.New("data fail"))

	req := PageRequest{Page: 1, PageSize: 10, SortBy: "name", SortDir: "asc"}
	allowed := map[string]string{"name": "name"}

	_, err := PaginatedSelect[testRow](context.Background(), db,
		"SELECT id, name, is_deleted FROM users", req, allowed)
	assert.Error(t, err)
}

// --- MustConnect ---

func TestMustConnectPanic(t *testing.T) {
	// MustConnect with an invalid DSN should panic
	assert.Panics(t, func() {
		MustConnect("postgres://invalid:invalid@localhost:1/nonexistent?sslmode=disable&connect_timeout=1",
			PoolConfig{MaxOpenConns: 1, MaxIdleConns: 1, ServiceName: "test"})
	})
}

func TestMustConnectSuccess(t *testing.T) {
	origOpen := openDB
	origRegister := registerDriver
	defer func() {
		openDB = origOpen
		registerDriver = origRegister
	}()

	registerOnce = sync.Once{}
	registered := false
	registerDriver = func(pool PoolConfig) {
		registered = true
		assert.Equal(t, "test-db", pool.ServiceName)
	}
	openDB = func(driverName, dsn string) (*sqlx.DB, error) {
		assert.Equal(t, "postgres-traced", driverName)
		assert.Equal(t, "sqlmock-success", dsn)
		rawDB, mock, err := sqlmock.NewWithDSN("sqlmock-success")
		require.NoError(t, err)
		mock.ExpectPing()
		t.Cleanup(func() { _ = rawDB.Close() })
		return sqlx.NewDb(rawDB, "sqlmock"), nil
	}

	db := MustConnect("sqlmock-success", PoolConfig{MaxOpenConns: 7, MaxIdleConns: 3, ServiceName: "test-db"})
	require.NotNil(t, db)
	assert.True(t, registered)
	assert.Equal(t, 7, db.Stats().MaxOpenConnections)
	assert.LessOrEqual(t, db.Stats().Idle, 3)
}

func TestMustConnectPanicOnOpenError(t *testing.T) {
	origOpen := openDB
	origRegister := registerDriver
	defer func() {
		openDB = origOpen
		registerDriver = origRegister
	}()

	registerOnce = sync.Once{}
	registerDriver = func(pool PoolConfig) {
		assert.Equal(t, "test", pool.ServiceName)
	}
	openDB = func(driverName, dsn string) (*sqlx.DB, error) {
		return nil, errors.New("open fail")
	}

	assert.PanicsWithValue(t, "database: open: open fail", func() {
		MustConnect("bad-dsn", PoolConfig{MaxOpenConns: 1, MaxIdleConns: 1, ServiceName: "test"})
	})
}

func TestMustConnectPanicOnPingError(t *testing.T) {
	origOpen := openDB
	origRegister := registerDriver
	defer func() {
		openDB = origOpen
		registerDriver = origRegister
	}()

	registerOnce = sync.Once{}
	registerDriver = func(pool PoolConfig) {
		assert.Equal(t, "test", pool.ServiceName)
	}
	openDB = func(driverName, dsn string) (*sqlx.DB, error) {
		rawDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		mock.ExpectPing().WillReturnError(errors.New("ping fail"))
		t.Cleanup(func() { _ = rawDB.Close() })
		return sqlx.NewDb(rawDB, "sqlmock"), nil
	}

	assert.PanicsWithValue(t, "database: ping: ping fail", func() {
		MustConnect("ping-dsn", PoolConfig{MaxOpenConns: 1, MaxIdleConns: 1, ServiceName: "test"})
	})
}
