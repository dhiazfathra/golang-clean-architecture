package featureflag

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/kvstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := kvstore.NewMockCache()
	svc := newServiceWithStore(repo, mc, 30*time.Second)
	return svc, mock
}

func TestIsEnabled_UnknownKey_ReturnsFalse(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(nil))

	assert.False(t, svc.IsEnabled("nonexistent"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsEnabled_L1_InProcessHit(t *testing.T) {
	svc, _ := newTestService(t)
	svc.store.Set(context.Background(), "my_flag", "1")

	assert.True(t, svc.IsEnabled("my_flag"))
}

func TestIsEnabled_L1_InProcessHit_Disabled(t *testing.T) {
	svc, _ := newTestService(t)
	svc.store.Set(context.Background(), "my_flag", "0")

	assert.False(t, svc.IsEnabled("my_flag"))
}

func TestCreate_PopulatesCacheAndReturnsFlag(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectExec(`INSERT INTO feature_flags`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	f, err := svc.Create(context.Background(), "new_flag", "a test flag", true, "user_1")
	require.NoError(t, err)
	assert.Equal(t, "new_flag", f.Key)
	assert.True(t, f.Enabled)
	assert.Equal(t, "a test flag", f.Description)
	assert.Equal(t, "user_1", f.CreatedBy)

	assert.True(t, svc.IsEnabled("new_flag"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_DisabledFlag(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectExec(`INSERT INTO feature_flags`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	f, err := svc.Create(context.Background(), "off_flag", "disabled", false, "user_1")
	require.NoError(t, err)
	assert.False(t, f.Enabled)
	assert.False(t, svc.IsEnabled("off_flag"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestToggle_FlipsCachedValue(t *testing.T) {
	svc, mock := newTestService(t)

	cols := flagColumns()
	rows := sqlmock.NewRows(cols).AddRow(flagRow(1, "toggle_me", true)...)
	mock.ExpectQuery(`SELECT \*`).WithArgs("toggle_me").WillReturnRows(rows)
	mock.ExpectExec(`UPDATE feature_flags`).WillReturnResult(sqlmock.NewResult(0, 1))

	err := svc.Toggle(context.Background(), "toggle_me", false, "user_1")
	require.NoError(t, err)
	assert.False(t, svc.IsEnabled("toggle_me"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestToggle_NotFound(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery(`SELECT \*`).WithArgs("missing").
		WillReturnRows(sqlmock.NewRows(nil))

	err := svc.Toggle(context.Background(), "missing", true, "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_RemovesFromCache(t *testing.T) {
	svc, mock := newTestService(t)
	svc.store.Set(context.Background(), "del_me", "1")

	cols := flagColumns()
	rows := sqlmock.NewRows(cols).AddRow(flagRow(42, "del_me", true)...)
	mock.ExpectQuery(`SELECT \*`).WithArgs("del_me").WillReturnRows(rows)
	mock.ExpectExec(`UPDATE feature_flags SET is_deleted`).WillReturnResult(sqlmock.NewResult(0, 1))

	err := svc.Delete(context.Background(), "del_me", "user_1")
	require.NoError(t, err)

	_, ok := svc.store.Local("del_me")
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_NotFound(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery(`SELECT \*`).WithArgs("missing").
		WillReturnRows(sqlmock.NewRows(nil))

	err := svc.Delete(context.Background(), "missing", "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_ReturnsFlags(t *testing.T) {
	svc, mock := newTestService(t)

	cols := flagColumns()
	rows := sqlmock.NewRows(cols).
		AddRow(flagRow(1, "flag_a", true)...).
		AddRow(flagRow(2, "flag_b", false)...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)

	flags, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, flags, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsEnabled_L2_CacheHit(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := kvstore.NewMockCache()
	mc.Data["ff:cached_flag"] = "1"
	svc := newServiceWithStore(repo, mc, 30*time.Second)

	assert.True(t, svc.IsEnabled("cached_flag"))
	v, ok := svc.store.Local("cached_flag")
	assert.True(t, ok)
	assert.Equal(t, "1", v)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsEnabled_L2_CacheHit_Disabled(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := kvstore.NewMockCache()
	mc.Data["ff:off_flag"] = "0"
	svc := newServiceWithStore(repo, mc, 30*time.Second)

	assert.False(t, svc.IsEnabled("off_flag"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsEnabled_L3_PostgresFallback(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := kvstore.NewMockCache()
	svc := newServiceWithStore(repo, mc, 30*time.Second)

	cols := flagColumns()
	rows := sqlmock.NewRows(cols).AddRow(flagRow(1, "db_flag", true)...)
	mock.ExpectQuery(`SELECT \*`).WithArgs("db_flag").WillReturnRows(rows)

	assert.True(t, svc.IsEnabled("db_flag"))
	v, ok := svc.store.Local("db_flag")
	assert.True(t, ok)
	assert.Equal(t, "1", v)
	val, err := mc.Get(context.Background(), "ff:db_flag")
	assert.NoError(t, err)
	assert.Equal(t, "1", val)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_DBError(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectExec(`INSERT INTO feature_flags`).
		WillReturnError(context.DeadlineExceeded)

	_, err := svc.Create(context.Background(), "fail_flag", "desc", true, "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestToggle_DBUpdateError(t *testing.T) {
	svc, mock := newTestService(t)
	cols := flagColumns()
	rows := sqlmock.NewRows(cols).AddRow(flagRow(1, "err_flag", true)...)
	mock.ExpectQuery(`SELECT \*`).WithArgs("err_flag").WillReturnRows(rows)
	mock.ExpectExec(`UPDATE feature_flags`).WillReturnError(context.DeadlineExceeded)

	err := svc.Toggle(context.Background(), "err_flag", false, "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestToggle_GenericDBError(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WithArgs("err_flag").
		WillReturnError(context.DeadlineExceeded)

	err := svc.Toggle(context.Background(), "err_flag", false, "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_DBDeleteError(t *testing.T) {
	svc, mock := newTestService(t)
	cols := flagColumns()
	rows := sqlmock.NewRows(cols).AddRow(flagRow(42, "err_flag", true)...)
	mock.ExpectQuery(`SELECT \*`).WithArgs("err_flag").WillReturnRows(rows)
	mock.ExpectExec(`UPDATE feature_flags SET is_deleted`).WillReturnError(context.DeadlineExceeded)

	err := svc.Delete(context.Background(), "err_flag", "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_GenericDBError(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WithArgs("err_flag").
		WillReturnError(context.DeadlineExceeded)

	err := svc.Delete(context.Background(), "err_flag", "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStartRefresh_RunsAndCancels(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := kvstore.NewMockCache()
	svc := newServiceWithStore(repo, mc, 50*time.Millisecond)

	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(flagColumns()))

	ctx, cancel := context.WithCancel(context.Background())
	svc.StartRefresh(ctx)
	time.Sleep(80 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestNewService_WrapsValkeyClient(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	repo := NewRepository(db)
	vk := testutil.SetupTestValkey(t)
	svc := NewService(repo, vk, 30*time.Second)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.store)
}

// --- helpers ---

func flagColumns() []string {
	return []string{
		"id", "key", "enabled", "description", "metadata",
		"created_at", "created_by", "updated_at", "updated_by", "is_deleted",
	}
}

func flagRow(id int64, key string, enabled bool) []driver.Value {
	now := time.Now()
	return []driver.Value{
		id, key, enabled, "test flag", []byte("{}"),
		now, "system", now, "system", false,
	}
}
