package apitoken

import (
	"context"
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

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/kvstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func echoCtx(method, target, body string) (echo.Context, *httptest.ResponseRecorder) {
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
	return c, rec
}

func newTestHandler(t *testing.T) (*Handler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := kvstore.NewMockCache()
	svc := newServiceWithStore(repo, mc, 30*time.Second)
	return NewHandler(svc), mock
}

func TestHandler_Create_OK(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectExec(`INSERT INTO api_tokens`).WillReturnResult(sqlmock.NewResult(1, 1))

	c, rec := echoCtx(http.MethodPost, "/admin/api-tokens",
		`{"name":"CI deploy","ttl_hours":720}`)

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp createTokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "gca_", resp.Token[:4])
	assert.Len(t, resp.Token, 68)
	assert.Equal(t, "CI deploy", resp.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_Create_MissingName(t *testing.T) {
	h, _ := newTestHandler(t)
	c, rec := echoCtx(http.MethodPost, "/admin/api-tokens",
		`{"ttl_hours":24}`)

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Create_InvalidTTL(t *testing.T) {
	h, _ := newTestHandler(t)
	c, rec := echoCtx(http.MethodPost, "/admin/api-tokens",
		`{"name":"test","ttl_hours":0}`)

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Create_BadBody(t *testing.T) {
	h, _ := newTestHandler(t)
	c, rec := echoCtx(http.MethodPost, "/admin/api-tokens", "{bad")

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Create_InternalError(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectExec(`INSERT INTO api_tokens`).
		WillReturnError(context.DeadlineExceeded)

	c, rec := echoCtx(http.MethodPost, "/admin/api-tokens",
		`{"name":"fail","ttl_hours":24}`)

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_List_OK(t *testing.T) {
	h, mock := newTestHandler(t)

	cols := tokenColumns()
	expires := time.Now().Add(time.Hour)
	rows := sqlmock.NewRows(cols).
		AddRow(tokenRow(1, "token_a", "hash_a", "gca_aaaa", "actor_1", expires)...).
		AddRow(tokenRow(2, "token_b", "hash_b", "gca_bbbb", "actor_1", expires)...)
	mock.ExpectQuery(`SELECT \*`).WithArgs("actor_1").WillReturnRows(rows)

	c, rec := echoCtx(http.MethodGet, "/admin/api-tokens", "")

	require.NoError(t, h.List(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var tokens []APIToken
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tokens))
	assert.Len(t, tokens, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_List_InternalError(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WithArgs("actor_1").
		WillReturnError(context.DeadlineExceeded)

	c, rec := echoCtx(http.MethodGet, "/admin/api-tokens", "")

	require.NoError(t, h.List(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_Revoke_OK(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectExec(`UPDATE api_tokens SET is_deleted`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	c, rec := echoCtx(http.MethodDelete, "/admin/api-tokens/42", "")
	c.SetParamNames("id")
	c.SetParamValues("42")

	require.NoError(t, h.Revoke(c))
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_Revoke_InvalidID(t *testing.T) {
	h, _ := newTestHandler(t)
	c, rec := echoCtx(http.MethodDelete, "/admin/api-tokens/abc", "")
	c.SetParamNames("id")
	c.SetParamValues("abc")

	require.NoError(t, h.Revoke(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Revoke_InternalError(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectExec(`UPDATE api_tokens SET is_deleted`).
		WillReturnError(context.DeadlineExceeded)

	c, rec := echoCtx(http.MethodDelete, "/admin/api-tokens/42", "")
	c.SetParamNames("id")
	c.SetParamValues("42")

	require.NoError(t, h.Revoke(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
