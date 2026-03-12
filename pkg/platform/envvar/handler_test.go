package envvar

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/logging"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func newTestHandler(t *testing.T) (*Handler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := newMockCache()
	svc := newServiceWithStore(repo, mc, 30*time.Second)
	return NewHandler(svc, logging.Noop()), mock
}

func TestHandler_CreateEnv_OK(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	mock.ExpectExec(`INSERT INTO env_vars`).WillReturnResult(sqlmock.NewResult(1, 1))

	c, rec := testutil.AuthedEchoCtx(http.MethodPost, "/envs",
		`{"platform":"mobile","key":"api_url","value":"https://api.example.com"}`)

	require.NoError(t, h.CreateEnv(c))
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp envResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_CreateEnv_MissingFields(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)
	c, rec := testutil.AuthedEchoCtx(http.MethodPost, "/envs",
		`{"platform":"mobile"}`)

	require.NoError(t, h.CreateEnv(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_CreateEnv_BadBody(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)
	c, rec := testutil.AuthedEchoCtx(http.MethodPost, "/envs", "{bad")

	require.NoError(t, h.CreateEnv(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_CreateEnv_PlatformTooLong(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)
	longPlatform := strings.Repeat("a", 31)
	c, rec := testutil.AuthedEchoCtx(http.MethodPost, "/envs",
		`{"platform":"`+longPlatform+`","key":"k","value":"v"}`)

	require.NoError(t, h.CreateEnv(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_CreateEnv_KeyTooLong(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)
	longKey := strings.Repeat("k", 51)
	c, rec := testutil.AuthedEchoCtx(http.MethodPost, "/envs",
		`{"platform":"mobile","key":"`+longKey+`","value":"v"}`)

	require.NoError(t, h.CreateEnv(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_CreateEnv_InternalError(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	mock.ExpectExec(`INSERT INTO env_vars`).WillReturnError(context.DeadlineExceeded)

	c, rec := testutil.AuthedEchoCtx(http.MethodPost, "/envs",
		`{"platform":"mobile","key":"k","value":"v"}`)

	require.NoError(t, h.CreateEnv(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_GetEnv_OK(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(1, "mobile", "api_url", "val")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)

	c, rec := testutil.AuthedEchoCtx(http.MethodGet, "/envs/mobile/api_url", "")
	c.SetParamNames("platform", "key")
	c.SetParamValues("mobile", "api_url")

	require.NoError(t, h.GetEnv(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_GetEnv_NotFound(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(nil))

	c, rec := testutil.AuthedEchoCtx(http.MethodGet, "/envs/mobile/missing", "")
	c.SetParamNames("platform", "key")
	c.SetParamValues("mobile", "missing")

	require.NoError(t, h.GetEnv(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_GetEnv_InternalError(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnError(context.DeadlineExceeded)

	c, rec := testutil.AuthedEchoCtx(http.MethodGet, "/envs/mobile/api_url", "")
	c.SetParamNames("platform", "key")
	c.SetParamValues("mobile", "api_url")

	require.NoError(t, h.GetEnv(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_GetEnvsByPlatform_OK(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)

	mock.ExpectQuery(`SELECT COUNT`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(1, "mobile", "api_url", "val")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)

	c, rec := testutil.AuthedEchoCtx(http.MethodGet, "/envs/mobile?page=1&page_size=10", "")
	c.SetParamNames("platform")
	c.SetParamValues("mobile")

	require.NoError(t, h.GetEnvsByPlatform(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_GetEnvsByPlatform_BindError(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)

	mock.ExpectQuery(`SELECT COUNT`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(envVarColumns()))

	// Send a JSON body with wrong types to trigger Bind error on GET
	c, rec := testutil.AuthedEchoCtx(http.MethodGet, "/envs/mobile", `{"page":"not_a_number"}`)
	c.SetParamNames("platform")
	c.SetParamValues("mobile")

	require.NoError(t, h.GetEnvsByPlatform(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_GetEnvsByPlatform_InternalError(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT COUNT`).WillReturnError(context.DeadlineExceeded)

	c, rec := testutil.AuthedEchoCtx(http.MethodGet, "/envs/mobile", "")
	c.SetParamNames("platform")
	c.SetParamValues("mobile")

	require.NoError(t, h.GetEnvsByPlatform(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_UpdateEnv_OK(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(1, "mobile", "api_url", "old")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)
	mock.ExpectExec(`UPDATE env_vars`).WillReturnResult(sqlmock.NewResult(0, 1))

	c, rec := testutil.AuthedEchoCtx(http.MethodPut, "/envs/mobile/api_url", `{"value":"new_val"}`)
	c.SetParamNames("platform", "key")
	c.SetParamValues("mobile", "api_url")

	require.NoError(t, h.UpdateEnv(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_UpdateEnv_NotFound(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(nil))

	c, rec := testutil.AuthedEchoCtx(http.MethodPut, "/envs/mobile/missing", `{"value":"val"}`)
	c.SetParamNames("platform", "key")
	c.SetParamValues("mobile", "missing")

	require.NoError(t, h.UpdateEnv(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_UpdateEnv_BadBody(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)
	c, rec := testutil.AuthedEchoCtx(http.MethodPut, "/envs/mobile/k", "{bad")
	c.SetParamNames("platform", "key")
	c.SetParamValues("mobile", "k")

	require.NoError(t, h.UpdateEnv(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_UpdateEnv_EmptyValue(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)
	c, rec := testutil.AuthedEchoCtx(http.MethodPut, "/envs/mobile/k", `{"value":""}`)
	c.SetParamNames("platform", "key")
	c.SetParamValues("mobile", "k")

	require.NoError(t, h.UpdateEnv(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_UpdateEnv_InternalError(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnError(context.DeadlineExceeded)

	c, rec := testutil.AuthedEchoCtx(http.MethodPut, "/envs/mobile/k", `{"value":"v"}`)
	c.SetParamNames("platform", "key")
	c.SetParamValues("mobile", "k")

	require.NoError(t, h.UpdateEnv(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_DeleteEnv_OK(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	cols := envVarColumns()
	rows := sqlmock.NewRows(cols).AddRow(envVarRow(42, "mobile", "del_key", "val")...)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(rows)
	mock.ExpectExec(`UPDATE env_vars SET is_deleted`).WillReturnResult(sqlmock.NewResult(0, 1))

	c, rec := testutil.AuthedEchoCtx(http.MethodDelete, "/envs/mobile/del_key", "")
	c.SetParamNames("platform", "key")
	c.SetParamValues("mobile", "del_key")

	require.NoError(t, h.DeleteEnv(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_DeleteEnv_NotFound(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnRows(sqlmock.NewRows(nil))

	c, rec := testutil.AuthedEchoCtx(http.MethodDelete, "/envs/mobile/missing", "")
	c.SetParamNames("platform", "key")
	c.SetParamValues("mobile", "missing")

	require.NoError(t, h.DeleteEnv(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandler_DeleteEnv_InternalError(t *testing.T) {
	t.Parallel()
	h, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT \*`).WillReturnError(context.DeadlineExceeded)

	c, rec := testutil.AuthedEchoCtx(http.MethodDelete, "/envs/mobile/k", "")
	c.SetParamNames("platform", "key")
	c.SetParamValues("mobile", "k")

	require.NoError(t, h.DeleteEnv(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
