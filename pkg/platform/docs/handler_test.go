package docs_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/docs"
)

func newTestFS() fs.FS {
	return fstest.MapFS{
		"scalar.html":  {Data: []byte("<html>scalar</html>")},
		"openapi.yaml": {Data: []byte("openapi: 3.0.3")},
	}
}

func TestScalarUI_OK(t *testing.T) {
	h := docs.NewHandler(newTestFS())
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	assert.NoError(t, h.ScalarUI(e.NewContext(req, rec)))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "scalar")
}

func TestOpenAPISpec_OK(t *testing.T) {
	h := docs.NewHandler(newTestFS())
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()
	assert.NoError(t, h.OpenAPISpec(e.NewContext(req, rec)))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "openapi")
}

func TestScalarUI_MissingFile(t *testing.T) {
	h := docs.NewHandler(fstest.MapFS{}) // empty FS
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	assert.NoError(t, h.ScalarUI(e.NewContext(req, rec)))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
