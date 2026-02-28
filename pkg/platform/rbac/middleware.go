package rbac

import (
	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/httputil"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

func RequirePermission(svc *Service, module, action string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := session.UserID(c)
			allowed, policy, err := svc.CheckPermission(c.Request().Context(), userID, module, action)
			if err != nil {
				return httputil.InternalError(c, err)
			}
			if !allowed {
				return c.JSON(403, map[string]string{"error": "forbidden"})
			}
			c.Set("rbac_field_policy", policy)
			return next(c)
		}
	}
}
