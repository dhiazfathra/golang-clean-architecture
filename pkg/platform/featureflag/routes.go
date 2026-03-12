package featureflag

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

// RegisterAdminRoutes registers feature flag management endpoints under /admin/feature-flags.
// All routes require the "featureflag:manage" RBAC permission.
func RegisterAdminRoutes(g *echo.Group, h *Handler, rbacSvc *rbac.Service, logger zerolog.Logger) {
	fg := g.Group("/feature-flags")
	fg.Use(rbac.RequirePermission(rbacSvc, logger, "featureflag", "manage"))
	fg.GET("", h.List)
	fg.POST("", h.Create)
	fg.PATCH("/:key", h.Toggle)
	fg.DELETE("/:key", h.Delete)
}
