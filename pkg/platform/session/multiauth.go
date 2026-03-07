package session

import (
	"context"
	"strings"

	"github.com/labstack/echo/v4"
)

// TokenValidator validates an opaque API token and returns the associated user ID.
type TokenValidator interface {
	Validate(ctx context.Context, rawToken string) (userID string, err error)
}

// RequireMultiAuth tries Bearer token first, then falls back to session cookie.
// On success, sets "user_id" and "auth_method" in the Echo context.
func RequireMultiAuth(store SessionStore, tokenSvc TokenValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Try Bearer token first
			if authHeader := c.Request().Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				rawToken := strings.TrimPrefix(authHeader, "Bearer ")
				userID, err := tokenSvc.Validate(c.Request().Context(), rawToken)
				if err != nil {
					return c.JSON(401, map[string]string{"error": "invalid token"})
				}
				c.Set("user_id", userID)
				c.Set("auth_method", "token")
				return next(c)
			}

			// Fall back to session cookie
			cookie, err := c.Cookie("session_id")
			if err != nil {
				return c.JSON(401, map[string]string{"error": "unauthorized"})
			}
			sess, err := store.Get(c.Request().Context(), cookie.Value)
			if err != nil || sess == nil {
				return c.JSON(401, map[string]string{"error": "unauthorized"})
			}
			c.Set("user_id", sess.UserID)
			c.Set("session", sess)
			c.Set("auth_method", "session")
			return next(c)
		}
	}
}

// AuthMethod returns the authentication method used for the current request.
// Returns "session" or "token".
func AuthMethod(c echo.Context) string {
	m, _ := c.Get("auth_method").(string)
	return m
}
