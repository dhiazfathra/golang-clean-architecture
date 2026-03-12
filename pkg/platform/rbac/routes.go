package rbac

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

func RegisterRoutes(adminGroup *echo.Group, h *Handler, svc *Service, logger zerolog.Logger) {
	rbacAdmin := adminGroup.Group("", RequirePermission(svc, logger, "rbac", "manage"))
	rbacAdmin.POST("/roles", h.CreateRole)
	rbacAdmin.GET("/roles", h.ListRoles)
	rbacAdmin.GET("/roles/:id", h.GetRole)
	rbacAdmin.DELETE("/roles/:id", h.DeleteRole)
	rbacAdmin.POST("/roles/:id/permissions", h.GrantPermission)
	rbacAdmin.DELETE("/roles/:id/permissions/:perm", h.RevokePermission)
	rbacAdmin.GET("/users/:id/roles", h.ListUserRoles)

	// Audit — separate permission so super_admin wildcard covers it without rbac/manage.
	adminGroup.GET("/audit/:type/:id", h.GetAuditHistory, RequirePermission(svc, logger, "audit", "read"))
}
