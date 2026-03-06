package envvar

import (
	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

// RegisterAdminRoutes registers env var management endpoints under /admin/envs.
// All routes require the "envvar:manage" RBAC permission.
func RegisterAdminRoutes(g *echo.Group, h *Handler, rbacSvc *rbac.Service) {
	eg := g.Group("/envs")
	eg.Use(rbac.RequirePermission(rbacSvc, "envvar", "manage"))
	eg.POST("", h.CreateEnv)
	eg.GET("/:platform/:key", h.GetEnv)
	eg.GET("/:platform", h.GetEnvsByPlatform)
	eg.PUT("/:platform/:key", h.UpdateEnv)
	eg.DELETE("/:platform/:key", h.DeleteEnv)
}
