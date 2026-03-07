package apitoken

import (
	"context"
	"database/sql/driver"
	"errors"
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

func tokenColumns() []string {
	return []string{
		"id", "name", "token_hash", "token_prefix", "user_id", "expires_at",
		"created_at", "created_by", "updated_at", "updated_by", "is_deleted",
	}
}

func tokenRow(id int64, name, hash, prefix, userID string, expiresAt time.Time) []driver.Value {
	now := time.Now()
	return []driver.Value{
		id, name, hash, prefix, userID, expiresAt,
		now, userID, now, userID, false,
	}
}

func TestCreate_ReturnsRawTokenAndStoresHash(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectExec(`INSERT INTO api_tokens`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	raw, token, err := svc.Create(context.Background(), "CI token", "user_1", 24*time.Hour)
	require.NoError(t, err)
	assert.True(t, len(raw) == 68, "raw token should be 68 chars (gca_ + 64 hex)")
	assert.Equal(t, "gca_", raw[:4])
	assert.Equal(t, "CI token", token.Name)
	assert.Equal(t, "user_1", token.UserID)
	assert.Equal(t, raw[:12], token.TokenPrefix)
	assert.NotEmpty(t, token.TokenHash)

	// Verify cache was populated
	v, ok := svc.store.Local(token.TokenHash)
	assert.True(t, ok)
	assert.Equal(t, "user_1", v)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_DBError(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectExec(`INSERT INTO api_tokens`).
		WillReturnError(context.DeadlineExceeded)

	_, _, err := svc.Create(context.Background(), "fail", "user_1", time.Hour)
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_TokenGenError(t *testing.T) {
	svc, _ := newTestService(t)
	svc.tokenFunc = func() (string, string, error) {
		return "", "", errors.New("rand failure")
	}

	_, _, err := svc.Create(context.Background(), "fail", "user_1", time.Hour)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generate token")
}

func TestValidate_L1Hit(t *testing.T) {
	svc, _ := newTestService(t)
	hash := hashToken("gca_testtoken123")
	svc.store.Set(context.Background(), hash, "user_42")

	userID, err := svc.Validate(context.Background(), "gca_testtoken123")
	require.NoError(t, err)
	assert.Equal(t, "user_42", userID)
}

func TestValidate_L2Hit(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := kvstore.NewMockCache()
	svc := newServiceWithStore(repo, mc, 30*time.Second)

	hash := hashToken("gca_l2token")
	mc.Data["apitoken:"+hash] = "user_l2" //nolint:secret_scan

	userID, err := svc.Validate(context.Background(), "gca_l2token")
	require.NoError(t, err)
	assert.Equal(t, "user_l2", userID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestValidate_L3Fallback(t *testing.T) {
	svc, mock := newTestService(t)

	raw := "gca_fallbacktoken1234567890abcdef1234567890abcdef1234567890abcdef"
	hash := hashToken(raw)
	expires := time.Now().Add(time.Hour)

	cols := tokenColumns()
	rows := sqlmock.NewRows(cols).AddRow(tokenRow(1, "test", hash, raw[:12], "user_db", expires)...)
	mock.ExpectQuery(`SELECT \*`).WithArgs(hash).WillReturnRows(rows)

	userID, err := svc.Validate(context.Background(), raw)
	require.NoError(t, err)
	assert.Equal(t, "user_db", userID)

	// Verify it was cached in L1
	v, ok := svc.store.Local(hash)
	assert.True(t, ok)
	assert.Equal(t, "user_db", v)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestValidate_ExpiredToken(t *testing.T) {
	svc, mock := newTestService(t)

	raw := "gca_expiredtoken1234567890abcdef1234567890abcdef1234567890abcde"
	hash := hashToken(raw)
	expires := time.Now().Add(-time.Hour) // expired

	cols := tokenColumns()
	rows := sqlmock.NewRows(cols).AddRow(tokenRow(1, "expired", hash, raw[:12], "user_exp", expires)...)
	mock.ExpectQuery(`SELECT \*`).WithArgs(hash).WillReturnRows(rows)

	_, err := svc.Validate(context.Background(), raw)
	require.Error(t, err)
	assert.Equal(t, "invalid token", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestValidate_InvalidToken(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(nil))

	_, err := svc.Validate(context.Background(), "gca_doesnotexist")
	require.Error(t, err)
	assert.Equal(t, "invalid token", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRevoke_SoftDeletes(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectExec(`UPDATE api_tokens SET is_deleted`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := svc.Revoke(context.Background(), 42, "user_1")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRevoke_DBError(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectExec(`UPDATE api_tokens SET is_deleted`).
		WillReturnError(context.DeadlineExceeded)

	err := svc.Revoke(context.Background(), 42, "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_ReturnsTokens(t *testing.T) {
	svc, mock := newTestService(t)

	cols := tokenColumns()
	expires := time.Now().Add(time.Hour)
	rows := sqlmock.NewRows(cols).
		AddRow(tokenRow(1, "token_a", "hash_a", "gca_aaaa", "user_1", expires)...).
		AddRow(tokenRow(2, "token_b", "hash_b", "gca_bbbb", "user_1", expires)...)
	mock.ExpectQuery(`SELECT \*`).WithArgs("user_1").WillReturnRows(rows)

	tokens, err := svc.List(context.Background(), "user_1")
	require.NoError(t, err)
	assert.Len(t, tokens, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_DBError(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectQuery(`SELECT \*`).WithArgs("user_1").
		WillReturnError(context.DeadlineExceeded)

	_, err := svc.List(context.Background(), "user_1")
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStartRefresh_RunsAndCancels(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := kvstore.NewMockCache()
	svc := newServiceWithStore(repo, mc, 50*time.Millisecond)

	mock.ExpectQuery(`SELECT token_hash, user_id`).
		WillReturnRows(sqlmock.NewRows([]string{"token_hash", "user_id"}))

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

func TestValidate_DBError(t *testing.T) {
	svc, mock := newTestService(t)

	mock.ExpectQuery(`SELECT \*`).
		WillReturnError(context.DeadlineExceeded)

	_, err := svc.Validate(context.Background(), "gca_dberror")
	require.Error(t, err)
	assert.Equal(t, "invalid token", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadAll_DBError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)

	mock.ExpectQuery(`SELECT token_hash, user_id`).
		WillReturnError(context.DeadlineExceeded)

	_, err := repo.LoadAll(context.Background())
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadAll_OK(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)

	rows := sqlmock.NewRows([]string{"token_hash", "user_id"}).
		AddRow("hash_a", "user_a").
		AddRow("hash_b", "user_b")
	mock.ExpectQuery(`SELECT token_hash, user_id`).WillReturnRows(rows)

	m, err := repo.LoadAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "user_a", m["hash_a"])
	assert.Equal(t, "user_b", m["hash_b"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHashToken_Deterministic(t *testing.T) {
	h1 := hashToken("gca_test123")
	h2 := hashToken("gca_test123")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64) // SHA-256 hex
}

func TestGenerateToken_Format(t *testing.T) {
	raw, hash, err := generateToken()
	require.NoError(t, err)
	assert.Equal(t, "gca_", raw[:4])
	assert.Len(t, raw, 68)
	assert.Len(t, hash, 64)
	assert.Equal(t, hashToken(raw), hash)
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("rand failure") }

func TestGenerateToken_RandError(t *testing.T) {
	orig := randReader
	randReader = errReader{}
	defer func() { randReader = orig }()

	_, _, err := generateToken()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rand failure")
}
