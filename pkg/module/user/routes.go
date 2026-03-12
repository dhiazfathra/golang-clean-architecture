package user

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func RegisterRoutes(g *echo.Group, h *Handler, rbacSvc *rbac.Service, logger zerolog.Logger) {
	r := g.Group("/users")
	r.POST("", h.Create, rbac.RequirePermission(rbacSvc, logger, "user", "create"))
	r.GET("/:id", h.GetByID, rbac.RequirePermission(rbacSvc, logger, "user", "read"))
	r.GET("", h.List, rbac.RequirePermission(rbacSvc, logger, "user", "list"))
	r.DELETE("/:id", h.Delete, rbac.RequirePermission(rbacSvc, logger, "user", "delete"))
}

// RegisterAdminRoutes mounts admin-only user endpoints under the admin group.
func RegisterAdminRoutes(adminGroup *echo.Group, h *Handler, rbacSvc *rbac.Service, logger zerolog.Logger) {
	adminGroup.GET("/users/:id", h.AdminGetByID, rbac.RequirePermission(rbacSvc, logger, "user", "read"))
}
