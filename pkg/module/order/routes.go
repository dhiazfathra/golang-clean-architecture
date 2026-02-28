package order

import (
	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func RegisterRoutes(g *echo.Group, h *Handler, rbacSvc *rbac.Service) {
	r := g.Group("/orders")
	r.POST("", h.Create, rbac.RequirePermission(rbacSvc, "order", "create"))
	r.GET("/:id", h.GetByID, rbac.RequirePermission(rbacSvc, "order", "read"))
	r.GET("", h.List, rbac.RequirePermission(rbacSvc, "order", "list"))
	r.DELETE("/:id", h.Delete, rbac.RequirePermission(rbacSvc, "order", "delete"))
}
