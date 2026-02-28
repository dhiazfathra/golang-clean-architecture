package auth

import "github.com/labstack/echo/v4"

func RegisterRoutes(public, protected *echo.Group, h *Handler) {
	public.POST("/auth/login", h.Login)
	protected.POST("/auth/logout", h.Logout)
	protected.GET("/auth/me", h.Me)
}
