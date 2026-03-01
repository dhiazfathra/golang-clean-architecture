package featureflag

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// RequireFlag returns an Echo middleware that returns 404 if the flag is not enabled.
// This keeps disabled features completely invisible to clients.
func RequireFlag(svc *Service, flagKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !svc.IsEnabled(flagKey) {
				return c.JSON(http.StatusNotFound, map[string]string{
					"error": "not found",
				})
			}
			return next(c)
		}
	}
}
