package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
)

func TestRegisterRoutes(t *testing.T) {
	e := echo.New()
	public := e.Group("")
	protected := e.Group("")

	mockHandler := &auth.Handler{}

	auth.RegisterRoutes(public, protected, mockHandler)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"Login route exists", http.MethodPost, "/auth/login", http.StatusBadRequest},
		{"Logout route exists", http.MethodPost, "/auth/logout", http.StatusOK},
		{"Me route exists", http.MethodGet, "/auth/me", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.NotEqual(t, http.StatusNotFound, rec.Code, "route should be registered")
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}
