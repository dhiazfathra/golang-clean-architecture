package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

// echoCtxWithPolicy creates an Echo context with a JSON body (may be empty) and injects
// the rbac AllFields policy so handler tests bypass RBAC filtering.
func echoCtxWithPolicy(method, target, body string) (echo.Context, *httptest.ResponseRecorder) {
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
	c.Set("rbac_field_policy", rbac.AllFields())
	return c, rec
}

// newHandlerWithMocks builds a Handler backed by a real Service with hand-rolled mocks.
func newHandlerWithMocks(repo ReadRepository, store *mockEventStore) *Handler {
	if store == nil {
		store = &mockEventStore{}
	}
	if repo == nil {
		repo = &mockReadRepo{}
	}
	svc := NewService(store, repo, &mockHasher{})
	return NewHandler(svc, nil)
}

// --- GetByID ---

func TestUserHandler_GetByID_OK(t *testing.T) {
	repo := &mockReadRepo{
		GetByIDFn: func(_ context.Context, _ string) (*UserReadModel, error) {
			return &UserReadModel{ID: 1, Email: "alice@example.com", Active: true}, nil
		},
	}
	h := newHandlerWithMocks(repo, nil)
	c, rec := echoCtxWithPolicy(http.MethodGet, "/users/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.GetByID(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "pass_hash")
	assert.NotContains(t, rec.Body.String(), "is_deleted")
}

func TestUserHandler_GetByID_NotFound(t *testing.T) {
	repo := &mockReadRepo{
		GetByIDFn: func(_ context.Context, _ string) (*UserReadModel, error) {
			return nil, nil
		},
	}
	h := newHandlerWithMocks(repo, nil)
	c, rec := echoCtxWithPolicy(http.MethodGet, "/users/999", "")
	c.SetParamNames("id")
	c.SetParamValues("999")

	require.NoError(t, h.GetByID(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Create ---

func TestUserHandler_Create_OK(t *testing.T) {
	repo := &mockReadRepo{
		GetByEmailFn: func(_ context.Context, _ string) (*UserReadModel, error) {
			return nil, nil
		},
	}
	h := newHandlerWithMocks(repo, nil)
	c, rec := echoCtxWithPolicy(http.MethodPost, "/users", `{"email":"bob@example.com","password":"secret"}`)

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["id"])
}

func TestUserHandler_Create_BadBody(t *testing.T) {
	h := newHandlerWithMocks(nil, nil)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("{bad json"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("rbac_field_policy", rbac.AllFields())

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_Create_DuplicateEmail_Returns500(t *testing.T) {
	repo := &mockReadRepo{
		GetByEmailFn: func(_ context.Context, _ string) (*UserReadModel, error) {
			return &UserReadModel{Email: "bob@example.com"}, nil
		},
	}
	h := newHandlerWithMocks(repo, nil)
	c, rec := echoCtxWithPolicy(http.MethodPost, "/users", `{"email":"bob@example.com","password":"secret"}`)

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- List ---

func TestUserHandler_List_OK(t *testing.T) {
	repo := &mockReadRepo{
		ListFn: func(_ context.Context, _ ListRequest) (*ListResponse, error) {
			return &ListResponse{
				Items:      []UserReadModel{{ID: 1, Email: "a@b.com"}},
				Total:      1,
				Page:       1,
				PageSize:   20,
				TotalPages: 1,
			}, nil
		},
	}
	h := newHandlerWithMocks(repo, nil)
	c, rec := echoCtxWithPolicy(http.MethodGet, "/users", "")

	require.NoError(t, h.List(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- Delete ---

func TestUserHandler_Delete_OK(t *testing.T) {
	h := newHandlerWithMocks(nil, nil)
	c, rec := echoCtxWithPolicy(http.MethodDelete, "/users/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.Delete(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- AdminGetByID ---

func TestUserHandler_AdminGetByID_OK(t *testing.T) {
	db, mock := testutil.NewMockDB(t)

	cols := []string{"id", "email", "pass_hash", "active", "created_at", "created_by", "updated_at", "updated_by", "is_deleted"}
	rows := sqlmock.NewRows(cols).
		AddRow(int64(1), "alice@example.com", "", true, time.Now(), "system", time.Now(), "system", true)
	mock.ExpectQuery(`SELECT \* FROM users_read WHERE id`).
		WithArgs("1").
		WillReturnRows(rows)

	svc := NewService(&mockEventStore{}, &mockReadRepo{}, &mockHasher{})
	h := NewHandler(svc, db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/admin/users/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Set("rbac_field_policy", rbac.AllFields())

	require.NoError(t, h.AdminGetByID(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserHandler_AdminGetByID_NotFound(t *testing.T) {
	db, mock := testutil.NewMockDB(t)

	mock.ExpectQuery(`SELECT \* FROM users_read WHERE id`).
		WithArgs("999").
		WillReturnError(sql.ErrNoRows)

	svc := NewService(&mockEventStore{}, &mockReadRepo{}, &mockHasher{})
	h := NewHandler(svc, db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/admin/users/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")
	c.Set("rbac_field_policy", rbac.AllFields())

	require.NoError(t, h.AdminGetByID(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
