package user

import (
	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func RegisterRoutes(g *echo.Group, h *Handler, rbacSvc *rbac.Service) {
	r := g.Group("/users")
	r.POST("", h.Create, rbac.RequirePermission(rbacSvc, "user", "create"))
	r.GET("/:id", h.GetByID, rbac.RequirePermission(rbacSvc, "user", "read"))
	r.GET("", h.List, rbac.RequirePermission(rbacSvc, "user", "list"))
	r.DELETE("/:id", h.Delete, rbac.RequirePermission(rbacSvc, "user", "delete"))
}
