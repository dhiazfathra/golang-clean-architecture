package featureflag

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
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/logging"
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
	return NewHandler(svc, logging.Noop()), mock
}

func TestHandler_Create_OK(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectExec(`INSERT INTO feature_flags`).WillReturnResult(sqlmock.NewResult(1, 1))

	c, rec := echoCtx(http.MethodPost, "/admin/feature-flags",
		`{"key":"new_flag","description":"test","enabled":true}`)

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp Flag
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "new_flag", resp.Key)
	assert.True(t, resp.Enabled)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_Create_MissingKey(t *testing.T) {
	h, _ := newTestHandler(t)
	c, rec := echoCtx(http.MethodPost, "/admin/feature-flags",
		`{"description":"no key"}`)

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Create_BadBody(t *testing.T) {
	h, _ := newTestHandler(t)
	c, rec := echoCtx(http.MethodPost, "/admin/feature-flags", "{bad")

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_List_OK(t *testing.T) {
	h, mock := newTestHandler(t)

	cols := flagColumns()
	rows := sqlmock.NewRows(cols).
		AddRow(flagRow(1, "flag_a", true)...).
		AddRow(flagRow(2, "flag_b", false)...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)

	c, rec := echoCtx(http.MethodGet, "/admin/feature-flags", "")

	require.NoError(t, h.List(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var flags []Flag
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &flags))
	assert.Len(t, flags, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_List_InternalError(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnError(context.DeadlineExceeded)

	c, rec := echoCtx(http.MethodGet, "/admin/feature-flags", "")

	require.NoError(t, h.List(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_Toggle_OK(t *testing.T) {
	h, mock := newTestHandler(t)

	cols := flagColumns()
	rows := sqlmock.NewRows(cols).AddRow(flagRow(1, "my_flag", true)...)
	mock.ExpectQuery(`SELECT \*`).WithArgs("my_flag").WillReturnRows(rows)
	mock.ExpectExec(`UPDATE feature_flags`).WillReturnResult(sqlmock.NewResult(0, 1))

	c, rec := echoCtx(http.MethodPatch, "/admin/feature-flags/my_flag", `{"enabled":false}`)
	c.SetParamNames("key")
	c.SetParamValues("my_flag")

	require.NoError(t, h.Toggle(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_Toggle_NotFound(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WithArgs("missing").
		WillReturnRows(sqlmock.NewRows(nil))

	c, rec := echoCtx(http.MethodPatch, "/admin/feature-flags/missing", `{"enabled":true}`)
	c.SetParamNames("key")
	c.SetParamValues("missing")

	require.NoError(t, h.Toggle(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_Toggle_BadBody(t *testing.T) {
	h, _ := newTestHandler(t)
	c, rec := echoCtx(http.MethodPatch, "/admin/feature-flags/x", "{bad")
	c.SetParamNames("key")
	c.SetParamValues("x")

	require.NoError(t, h.Toggle(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Delete_OK(t *testing.T) {
	h, mock := newTestHandler(t)

	cols := flagColumns()
	rows := sqlmock.NewRows(cols).AddRow(flagRow(42, "del_flag", true)...)
	mock.ExpectQuery(`SELECT \*`).WithArgs("del_flag").WillReturnRows(rows)
	mock.ExpectExec(`UPDATE feature_flags SET is_deleted`).WillReturnResult(sqlmock.NewResult(0, 1))

	c, rec := echoCtx(http.MethodDelete, "/admin/feature-flags/del_flag", "")
	c.SetParamNames("key")
	c.SetParamValues("del_flag")

	require.NoError(t, h.Delete(c))
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_Create_InternalError(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectExec(`INSERT INTO feature_flags`).
		WillReturnError(context.DeadlineExceeded)

	c, rec := echoCtx(http.MethodPost, "/admin/feature-flags",
		`{"key":"fail","enabled":true}`)

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_Toggle_InternalError(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WithArgs("err").
		WillReturnError(context.DeadlineExceeded)

	c, rec := echoCtx(http.MethodPatch, "/admin/feature-flags/err", `{"enabled":true}`)
	c.SetParamNames("key")
	c.SetParamValues("err")

	require.NoError(t, h.Toggle(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_Delete_InternalError(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WithArgs("err").
		WillReturnError(context.DeadlineExceeded)

	c, rec := echoCtx(http.MethodDelete, "/admin/feature-flags/err", "")
	c.SetParamNames("key")
	c.SetParamValues("err")

	require.NoError(t, h.Delete(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_Delete_NotFound(t *testing.T) {
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WithArgs("missing").
		WillReturnRows(sqlmock.NewRows(nil))

	c, rec := echoCtx(http.MethodDelete, "/admin/feature-flags/missing", "")
	c.SetParamNames("key")
	c.SetParamValues("missing")

	require.NoError(t, h.Delete(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
