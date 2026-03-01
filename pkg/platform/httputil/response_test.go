package httputil

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/observability"
)

func init() {
	observability.InitNoop()
}

func newTestContext(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestOK(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/", "")
	err := OK(c, map[string]string{"msg": "hello"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "hello")
}

func TestCreated(t *testing.T) {
	c, rec := newTestContext(http.MethodPost, "/", "")
	err := Created(c, map[string]string{"id": "1"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "1")
}

func TestBadRequest(t *testing.T) {
	c, rec := newTestContext(http.MethodPost, "/", "")
	err := BadRequest(c, "bad input")
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "bad input")
}

func TestInternalError(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/", "")
	err := InternalError(c, errors.New("boom"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "internal server error")
}

func TestNotFound(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/", "")
	err := NotFound(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "not found")
}

func TestNotFoundOrError_NotFound(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/", "")
	err := NotFoundOrError(c, ErrNotFound)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestNotFoundOrError_InternalError(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/", "")
	err := NotFoundOrError(c, errors.New("db fail"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

type testBody struct {
	Name string `json:"name" validate:"required"`
}

type testValidator struct{}

func (tv *testValidator) Validate(i any) error {
	b, ok := i.(*testBody)
	if !ok {
		return errors.New("invalid type")
	}
	if b.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func TestBindAndValidate_Success(t *testing.T) {
	c, _ := newTestContext(http.MethodPost, "/", `{"name":"Alice"}`)
	c.Echo().Validator = &testValidator{}
	var body testBody
	err := BindAndValidate(c, &body)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", body.Name)
}

func TestBindAndValidate_InvalidJSON(t *testing.T) {
	c, rec := newTestContext(http.MethodPost, "/", `{invalid`)
	c.Echo().Validator = &testValidator{}
	var body testBody
	err := BindAndValidate(c, &body)
	assert.NoError(t, err) // BadRequest returns nil error, writes to response
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBindAndValidate_ValidationFail(t *testing.T) {
	c, rec := newTestContext(http.MethodPost, "/", `{"name":""}`)
	c.Echo().Validator = &testValidator{}
	var body testBody
	err := BindAndValidate(c, &body)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "name is required")
}
