package docs

import (
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Handler serves the Scalar API reference UI and the OpenAPI spec.
// Construct with an fs.FS rooted at the directory containing
// "scalar.html" and "openapi.yaml" (typically the embedded api.Files).
type Handler struct {
	files fs.FS
}

func NewHandler(files fs.FS) *Handler {
	return &Handler{files: files}
}

// ScalarUI handles GET /docs — serves the Scalar HTML page.
func (h *Handler) ScalarUI(c echo.Context) error {
	content, err := fs.ReadFile(h.files, "scalar.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, "docs unavailable")
	}
	return c.HTMLBlob(http.StatusOK, content)
}

// OpenAPISpec handles GET /openapi.yaml — serves the raw spec.
func (h *Handler) OpenAPISpec(c echo.Context) error {
	content, err := fs.ReadFile(h.files, "openapi.yaml")
	if err != nil {
		return c.String(http.StatusInternalServerError, "spec unavailable")
	}
	return c.Blob(http.StatusOK, "application/yaml", content)
}
