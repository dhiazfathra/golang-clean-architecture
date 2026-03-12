package order

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func RegisterRoutes(g *echo.Group, h *Handler, rbacSvc *rbac.Service, logger zerolog.Logger) {
	r := g.Group("/orders")
	r.POST("", h.Create, rbac.RequirePermission(rbacSvc, logger, "order", "create"))
	r.GET("/:id", h.GetByID, rbac.RequirePermission(rbacSvc, logger, "order", "read"))
	r.GET("", h.List, rbac.RequirePermission(rbacSvc, logger, "order", "list"))
	r.DELETE("/:id", h.Delete, rbac.RequirePermission(rbacSvc, logger, "order", "delete"))
}
