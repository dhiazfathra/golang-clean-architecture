package session

import (
	"github.com/labstack/echo/v4"
)

func RequireSession(store SessionStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
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
			return next(c)
		}
	}
}

func UserID(c echo.Context) string {
	id, _ := c.Get("user_id").(string)
	return id
}
