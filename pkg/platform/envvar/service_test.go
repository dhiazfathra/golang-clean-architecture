package envvar

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := newMockCache()
	svc := newServiceWithCache(repo, mc, 30*time.Second)
	return svc, mock
}

func TestGetValue_UnknownKey_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(nil))

	assert.Equal(t, "", svc.GetValue("mobile", "nonexistent"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetValue_L1_InProcessHit(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	svc.local.Store("mobile:api_url", "https://api.example.com")

	assert.Equal(t, "https://api.example.com", svc.GetValue("mobile", "api_url"))
}

func TestGetValue_L2_CacheHit(t *testing.T) {
	t.Parallel()
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := newMockCache()
	mc.data["env:mobile:api_url"] = "https://cached.example.com"
	svc := newServiceWithCache(repo, mc, 30*time.Second)

	assert.Equal(t, "https://cached.example.com", svc.GetValue("mobile", "api_url"))
	v, ok := svc.local.Load("mobile:api_url")
	assert.True(t, ok)
	assert.Equal(t, "https://cached.example.com", v.(string))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetValue_L3_PostgresFallback(t *testing.T) {
	t.Parallel()
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := newMockCache()
	svc := newServiceWithCache(repo, mc, 30*time.Second)

	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(1, "mobile", "api_url", "https://db.example.com")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)

	assert.Equal(t, "https://db.example.com", svc.GetValue("mobile", "api_url"))
	v, ok := svc.local.Load("mobile:api_url")
	assert.True(t, ok)
	assert.Equal(t, "https://db.example.com", v.(string))
	val, err := mc.Get(context.Background(), "env:mobile:api_url")
	assert.NoError(t, err)
	assert.Equal(t, "https://db.example.com", val)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGet_OK(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(1, "mobile", "api_url", "val")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)

	e, err := svc.Get(context.Background(), "mobile", "api_url")
	require.NoError(t, err)
	assert.Equal(t, "mobile", e.Platform)
	assert.Equal(t, "api_url", e.Key)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(nil))

	_, err := svc.Get(context.Background(), "mobile", "missing")
	require.Error(t, err)
	assert.Equal(t, "env var not found", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGet_DBError(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnError(context.DeadlineExceeded)

	_, err := svc.Get(context.Background(), "mobile", "api_url")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_OK(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	mock.ExpectExec(`INSERT INTO env_vars`).WillReturnResult(sqlmock.NewResult(1, 1))

	e, err := svc.Create(context.Background(), "mobile", "api_url", "https://api.example.com", "user_1")
	require.NoError(t, err)
	assert.Equal(t, "mobile", e.Platform)
	assert.Equal(t, "api_url", e.Key)
	assert.Equal(t, "https://api.example.com", e.Value)
	assert.Equal(t, "user_1", e.CreatedBy)

	assert.Equal(t, "https://api.example.com", svc.GetValue("mobile", "api_url"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_DBError(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	mock.ExpectExec(`INSERT INTO env_vars`).WillReturnError(context.DeadlineExceeded)

	_, err := svc.Create(context.Background(), "mobile", "api_url", "val", "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate_OK(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(1, "mobile", "api_url", "old_val")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)
	mock.ExpectExec(`UPDATE env_vars`).WillReturnResult(sqlmock.NewResult(0, 1))

	e, err := svc.Update(context.Background(), "mobile", "api_url", "new_val", "user_1")
	require.NoError(t, err)
	assert.Equal(t, "new_val", e.Value)
	assert.Equal(t, "new_val", svc.GetValue("mobile", "api_url"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate_NotFound(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(nil))

	_, err := svc.Update(context.Background(), "mobile", "missing", "val", "user_1")
	require.Error(t, err)
	assert.Equal(t, "env var not found", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate_DBUpdateError(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(1, "mobile", "api_url", "old")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)
	mock.ExpectExec(`UPDATE env_vars`).WillReturnError(context.DeadlineExceeded)

	_, err := svc.Update(context.Background(), "mobile", "api_url", "new", "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate_GenericDBError(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnError(context.DeadlineExceeded)

	_, err := svc.Update(context.Background(), "mobile", "api_url", "val", "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_OK(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	svc.local.Store("mobile:del_me", "val")

	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(42, "mobile", "del_me", "val")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)
	mock.ExpectExec(`UPDATE env_vars SET is_deleted`).WillReturnResult(sqlmock.NewResult(0, 1))

	err := svc.Delete(context.Background(), "mobile", "del_me", "user_1")
	require.NoError(t, err)

	_, ok := svc.local.Load("mobile:del_me")
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_NotFound(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(nil))

	err := svc.Delete(context.Background(), "mobile", "missing", "user_1")
	require.Error(t, err)
	assert.Equal(t, "env var not found", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_DBDeleteError(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(42, "mobile", "del_me", "val")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)
	mock.ExpectExec(`UPDATE env_vars SET is_deleted`).WillReturnError(context.DeadlineExceeded)

	err := svc.Delete(context.Background(), "mobile", "del_me", "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_GenericDBError(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnError(context.DeadlineExceeded)

	err := svc.Delete(context.Background(), "mobile", "del_me", "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReload_PopulatesLocalCache(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).
		AddRow(envVarRow(1, "mobile", "api_url", "https://m.example.com")...).
		AddRow(envVarRow(2, "web", "api_url", "https://w.example.com")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)

	err := svc.reload(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "https://m.example.com", svc.GetValue("mobile", "api_url"))
	assert.Equal(t, "https://w.example.com", svc.GetValue("web", "api_url"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReload_PrunesStaleKeys(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	svc.local.Store("mobile:stale_key", "old_val")

	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(1, "mobile", "fresh_key", "new_val")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)

	err := svc.reload(context.Background())
	require.NoError(t, err)

	_, ok := svc.local.Load("mobile:stale_key")
	assert.False(t, ok, "stale key should be pruned")
	assert.Equal(t, "new_val", svc.GetValue("mobile", "fresh_key"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReload_DBError(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnError(context.DeadlineExceeded)

	err := svc.reload(context.Background())
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReload_EmptyEnvVars(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(envVarColumns()))

	err := svc.reload(context.Background())
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStartRefresh_RunsAndCancels(t *testing.T) {
	t.Parallel()
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := newMockCache()
	svc := newServiceWithCache(repo, mc, 50*time.Millisecond)

	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(envVarColumns()))

	ctx, cancel := context.WithCancel(context.Background())
	svc.StartRefresh(ctx)
	time.Sleep(80 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestNewService_WrapsValkeyClient(t *testing.T) {
	t.Parallel()
	db, _ := testutil.NewMockDB(t)
	repo := NewRepository(db)
	vk := testutil.SetupTestValkey(t)
	svc := NewService(repo, vk, 30*time.Second)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.cache)
}

func TestListByPlatform_OK(t *testing.T) {
	t.Parallel()
	svc, mock := newTestService(t)

	// Count query
	mock.ExpectQuery(`SELECT COUNT`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// Data query
	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(1, "mobile", "api_url", "val")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)

	req := database.PageRequest{Page: 1, PageSize: 10, SortBy: "key", SortDir: "asc"}
	page, err := svc.ListByPlatform(context.Background(), "mobile", req)
	require.NoError(t, err)
	assert.Len(t, page.Items, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- helpers ---

func envVarColumns() []string {
	return []string{
		"id", "platform", "key", "value",
		"created_at", "created_by", "updated_at", "updated_by", "is_deleted",
	}
}

func envVarRow(id int64, platform, key, value string) []driver.Value {
	now := time.Now()
	return []driver.Value{
		id, platform, key, value,
		now, "system", now, "system", false,
	}
}
