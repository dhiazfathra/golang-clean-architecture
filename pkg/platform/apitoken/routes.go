package apitoken

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func RegisterRoutes(g *echo.Group, h *Handler, rbacSvc *rbac.Service, logger zerolog.Logger) {
	tg := g.Group("/api-tokens")
	tg.Use(rbac.RequirePermission(rbacSvc, logger, "apitoken", "manage"))
	tg.POST("", h.Create)
	tg.GET("", h.List)
	tg.DELETE("/:id", h.Revoke)
}
