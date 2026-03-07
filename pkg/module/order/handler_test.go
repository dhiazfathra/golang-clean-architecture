package order

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func newTestHandler(repo ReadRepository, store *mockEventStore, userProv UserProvider) *Handler {
	if store == nil {
		store = &mockEventStore{}
	}
	if repo == nil {
		repo = &mockReadRepo{}
	}
	if userProv == nil {
		userProv = &mockUserProvider{}
	}
	return NewHandler(NewService(store, repo, userProv))
}

// --- Create ---

func TestOrderHandler_Create_OK(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodPost, "/orders",
		`{"user_id":"100","total":49.99}`, rbac.AllFields())

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["id"])
}

func TestOrderHandler_Create_BadBody(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader("{bad"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("rbac_field_policy", rbac.AllFields())

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOrderHandler_Create_UserNotFound_Returns500(t *testing.T) {
	userProv := &mockUserProvider{
		GetByIDFn: func(_ context.Context, _ string) (bool, error) { return false, nil },
	}
	h := newTestHandler(nil, nil, userProv)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodPost, "/orders", `{"user_id":"999","total":1.0}`, rbac.AllFields())

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- GetByID ---

func TestOrderHandler_GetByID_OK(t *testing.T) {
	repo := &mockReadRepo{
		GetByIDFn: func(_ context.Context, _ string) (*OrderReadModel, error) {
			return &OrderReadModel{ID: 1, Status: "pending"}, nil
		},
	}
	h := newTestHandler(repo, nil, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodGet, "/orders/1", "", rbac.AllFields())
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.GetByID(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestOrderHandler_GetByID_NotFound(t *testing.T) {
	repo := &mockReadRepo{
		GetByIDFn: func(_ context.Context, _ string) (*OrderReadModel, error) {
			return nil, nil
		},
	}
	h := newTestHandler(repo, nil, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodGet, "/orders/999", "", rbac.AllFields())
	c.SetParamNames("id")
	c.SetParamValues("999")

	require.NoError(t, h.GetByID(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- List ---

func TestOrderHandler_List_OK(t *testing.T) {
	repo := &mockReadRepo{
		ListFn: func(_ context.Context, _ ListRequest) (*ListResponse, error) {
			return &ListResponse{Items: []OrderReadModel{{ID: 1}}, Total: 1}, nil
		},
	}
	h := newTestHandler(repo, nil, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodGet, "/orders", "", rbac.AllFields())

	require.NoError(t, h.List(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- Delete ---

func TestOrderHandler_Delete_OK(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodDelete, "/orders/1", "", rbac.AllFields())
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.Delete(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}
