package rbac_test

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
	now          = time.Now()
	roleCols     = []string{"id", "name", "description", "created_at", "updated_at"}
	permCols     = []string{"id", "role_id", "module", "action", "field_mode", "field_list", "created_at", "updated_at"}
	userRoleCols = []string{"user_id", "role_id", "created_at", "updated_at"}
)

func mockRoleRow(id int64, name string) *sqlmock.Rows {
	return sqlmock.NewRows(roleCols).AddRow(id, name, "desc", now, now)
}

func mockPermRow(id, roleID int64, module, action string) *sqlmock.Rows {
	return sqlmock.NewRows(permCols).AddRow(id, roleID, module, action, "all", pq.StringArray{}, now, now)
}

// --- NewPgReadRepository ---

func TestNewPgReadRepository(t *testing.T) {
	db, _ := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)
	assert.NotNil(t, repo)
}

// --- GetRoleByID ---

func TestGetRoleByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM roles_read WHERE id = $1`)).
		WithArgs("role-1").
		WillReturnRows(mockRoleRow(1, "admin"))

	got, err := repo.GetRoleByID(context.Background(), "role-1")
	require.NoError(t, err)
	assert.Equal(t, "admin", got.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRoleByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM roles_read WHERE id = $1`)).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows(roleCols))

	got, err := repo.GetRoleByID(context.Background(), "missing")
	assert.Nil(t, got)
	_ = err
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRoleByID_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM roles_read WHERE id = $1`)).
		WithArgs("role-1").
		WillReturnError(errors.New("db error"))

	got, err := repo.GetRoleByID(context.Background(), "role-1")
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetRoleByName ---

func TestGetRoleByName_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM roles_read WHERE name = $1`)).
		WithArgs("admin").
		WillReturnRows(mockRoleRow(1, "admin"))

	got, err := repo.GetRoleByName(context.Background(), "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", got.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRoleByName_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM roles_read WHERE name = $1`)).
		WithArgs("ghost").
		WillReturnRows(sqlmock.NewRows(roleCols))

	got, err := repo.GetRoleByName(context.Background(), "ghost")
	assert.Nil(t, got)
	_ = err
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRoleByName_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM roles_read WHERE name = $1`)).
		WithArgs("admin").
		WillReturnError(errors.New("db error"))

	got, err := repo.GetRoleByName(context.Background(), "admin")
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- ListRoles ---

func TestListRoles_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM roles_read ORDER BY name ASC`)).
		WillReturnRows(sqlmock.NewRows(roleCols).
			AddRow(int64(1), "admin", "desc", now, now).
			AddRow(int64(2), "viewer", "desc", now, now))

	got, err := repo.ListRoles(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "admin", got[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListRoles_Empty(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM roles_read ORDER BY name ASC`)).
		WillReturnRows(sqlmock.NewRows(roleCols))

	got, err := repo.ListRoles(context.Background())
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListRoles_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM roles_read ORDER BY name ASC`)).
		WillReturnError(errors.New("db error"))

	got, err := repo.ListRoles(context.Background())
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetPermissionsForRole ---

func TestGetPermissionsForRole_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM permissions_read WHERE role_id = $1`)).
		WithArgs("role-1").
		WillReturnRows(mockPermRow(1, 1, "users", "read"))

	got, err := repo.GetPermissionsForRole(context.Background(), "role-1")
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "read", got[0].Action)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPermissionsForRole_Empty(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM permissions_read WHERE role_id = $1`)).
		WithArgs("role-99").
		WillReturnRows(sqlmock.NewRows(permCols))

	got, err := repo.GetPermissionsForRole(context.Background(), "role-99")
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPermissionsForRole_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM permissions_read WHERE role_id = $1`)).
		WithArgs("role-1").
		WillReturnError(errors.New("db error"))

	got, err := repo.GetPermissionsForRole(context.Background(), "role-1")
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetRolesForUser ---

func TestGetRolesForUser_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM user_roles_read WHERE user_id = $1`)).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows(userRoleCols).
			AddRow(int64(1), int64(10), now, now).
			AddRow(int64(1), int64(20), now, now))

	got, err := repo.GetRolesForUser(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"10", "20"}, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRolesForUser_Empty(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM user_roles_read WHERE user_id = $1`)).
		WithArgs("user-99").
		WillReturnRows(sqlmock.NewRows(userRoleCols))

	got, err := repo.GetRolesForUser(context.Background(), "user-99")
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRolesForUser_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := rbac.NewPgReadRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM user_roles_read WHERE user_id = $1`)).
		WithArgs("user-1").
		WillReturnError(errors.New("db error"))

	got, err := repo.GetRolesForUser(context.Background(), "user-1")
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
