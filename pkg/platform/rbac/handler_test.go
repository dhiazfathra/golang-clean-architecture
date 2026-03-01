package rbac

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func rbacEchoCtx(method, target, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "actor_1")
	c.Set("rbac_field_policy", AllFields())
	return c, rec
}

// --- CreateRole handler ---

func TestHandler_CreateRole_OK(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{})
	h := NewHandler(svc, nil)
	c, rec := rbacEchoCtx(http.MethodPost, "/admin/roles",
		`{"name":"editor","description":"Can edit","permissions":[]}`)

	require.NoError(t, h.CreateRole(c))
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestHandler_CreateRole_EmptyName(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{})
	h := NewHandler(svc, nil)
	c, rec := rbacEchoCtx(http.MethodPost, "/admin/roles", `{"name":""}`)

	require.NoError(t, h.CreateRole(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ListRoles handler ---

func TestHandler_ListRoles_OK(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{
		ListRolesFn: func(_ context.Context) ([]RoleReadModel, error) {
			return []RoleReadModel{{ID: 1, Name: "admin"}}, nil
		},
	})
	h := NewHandler(svc, nil)
	c, rec := rbacEchoCtx(http.MethodGet, "/admin/roles", "")

	require.NoError(t, h.ListRoles(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "admin")
}

// --- GetRole handler ---

func TestHandler_GetRole_Found(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{
		GetRoleByIDFn: func(_ context.Context, _ string) (*RoleReadModel, error) {
			return &RoleReadModel{ID: 1, Name: "admin"}, nil
		},
		GetPermissionsForRoleFn: func(_ context.Context, _ string) ([]PermissionReadModel, error) {
			return nil, nil
		},
	})
	h := NewHandler(svc, nil)
	c, rec := rbacEchoCtx(http.MethodGet, "/admin/roles/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.GetRole(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_GetRole_NotFound(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{
		GetRoleByIDFn: func(_ context.Context, _ string) (*RoleReadModel, error) {
			return nil, nil
		},
	})
	h := NewHandler(svc, nil)
	c, rec := rbacEchoCtx(http.MethodGet, "/admin/roles/999", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	require.NoError(t, h.GetRole(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- DeleteRole handler ---

func TestHandler_DeleteRole_OK(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{})
	h := NewHandler(svc, nil)
	c, rec := rbacEchoCtx(http.MethodDelete, "/admin/roles/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.DeleteRole(c))
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// --- GrantPermission handler ---

func TestHandler_GrantPermission_OK(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{})
	h := NewHandler(svc, nil)
	c, rec := rbacEchoCtx(http.MethodPost, "/admin/roles/1/permissions",
		`{"module":"orders","action":"read","fields":{"mode":"all"}}`)
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.GrantPermission(c))
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// --- RevokePermission handler ---

func TestHandler_RevokePermission_OK(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{})
	h := NewHandler(svc, nil)
	c, rec := rbacEchoCtx(http.MethodDelete, "/admin/roles/1/permissions/orders:read", "")
	c.SetParamNames("id", "perm")
	c.SetParamValues("1", "orders:read")

	require.NoError(t, h.RevokePermission(c))
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHandler_RevokePermission_BadPerm(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{})
	h := NewHandler(svc, nil)
	c, rec := rbacEchoCtx(http.MethodDelete, "/admin/roles/1/permissions/bad", "")
	c.SetParamNames("id", "perm")
	c.SetParamValues("1", "bad")

	require.NoError(t, h.RevokePermission(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ListUserRoles handler ---

func TestHandler_ListUserRoles_OK(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"role_1", "role_2"}, nil
		},
	})
	h := NewHandler(svc, nil)
	c, rec := rbacEchoCtx(http.MethodGet, "/admin/users/u1/roles", "")
	c.SetParamNames("id")
	c.SetParamValues("u1")

	require.NoError(t, h.ListUserRoles(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "role_1")
}

// --- GetAuditHistory handler ---

func TestHandler_GetAuditHistory_OK(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	cols := []string{"id", "event_type", "version", "data", "metadata", "created_at"}
	rows := sqlmock.NewRows(cols).
		AddRow(int64(1), "user.created", 1, []byte(`{}`), []byte(`{}`), time.Now())
	mock.ExpectQuery(`SELECT id, event_type, version, data, metadata, created_at`).
		WithArgs("user", "usr_1").
		WillReturnRows(rows)

	svc := NewService(&mockEventStore{}, &mockRepo{})
	h := NewHandler(svc, db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/admin/audit/user/usr_1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("type", "id")
	c.SetParamValues("user", "usr_1")

	require.NoError(t, h.GetAuditHistory(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "user.created")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_GetAuditHistory_DBError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	mock.ExpectQuery(`SELECT id, event_type`).
		WithArgs("user", "bad_id").
		WillReturnError(assert.AnError)

	svc := NewService(&mockEventStore{}, &mockRepo{})
	h := NewHandler(svc, db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/admin/audit/user/bad_id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("type", "id")
	c.SetParamValues("user", "bad_id")

	require.NoError(t, h.GetAuditHistory(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- parsePermParam ---

func TestParsePermParam_Valid(t *testing.T) {
	mod, act := parsePermParam("orders:read")
	assert.Equal(t, "orders", mod)
	assert.Equal(t, "read", act)
}

func TestParsePermParam_Invalid(t *testing.T) {
	mod, act := parsePermParam("badparam")
	assert.Empty(t, mod)
	assert.Empty(t, act)
}
