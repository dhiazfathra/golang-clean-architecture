package envvar

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

// RegisterRoutes registers env var management endpoints under /envs.
// All routes require the "envvar:manage" RBAC permission.
func RegisterRoutes(g *echo.Group, h *Handler, rbacSvc *rbac.Service, logger zerolog.Logger) {
	eg := g.Group("/envs")
	eg.Use(rbac.RequirePermission(rbacSvc, logger, "envvar", "manage"))
	eg.POST("", h.CreateEnv)
	eg.GET("/:platform/:key", h.GetEnv)
	eg.GET("/:platform", h.GetEnvsByPlatform)
	eg.PUT("/:platform/:key", h.UpdateEnv)
	eg.DELETE("/:platform/:key", h.DeleteEnv)
}
