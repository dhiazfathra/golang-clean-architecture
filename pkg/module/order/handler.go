package order

import (
	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/httputil"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type createOrderReq struct {
	UserID string  `json:"user_id"`
	Total  float64 `json:"total"`
}

func (h *Handler) Create(c echo.Context) error {
	var req createOrderReq
	if err := c.Bind(&req); err != nil {
		return httputil.BadRequest(c, "invalid body")
	}
	actor := session.UserID(c)
	id, err := h.svc.CreateOrder(c.Request().Context(), CreateOrderCmd{
		UserID: req.UserID,
		Total:  req.Total,
		Actor:  actor,
	})
	if err != nil {
		return httputil.InternalError(c, err)
	}
	return httputil.Created(c, map[string]string{"id": id})
}

func (h *Handler) GetByID(c echo.Context) error {
	o, err := h.svc.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil || o == nil {
		return httputil.NotFound(c)
	}
	return httputil.OK(c, rbac.FilterResponse(c, o))
}

func (h *Handler) List(c echo.Context) error {
	req := ListRequest{Page: 1, PageSize: 20, SortBy: "created_at", SortDir: "asc"}
	if err := c.Bind(&req); err != nil {
		return httputil.BadRequest(c, "invalid query")
	}
	page, err := h.svc.List(c.Request().Context(), req)
	if err != nil {
		return httputil.InternalError(c, err)
	}
	return httputil.OK(c, rbac.FilterResponse(c, page))
}

func (h *Handler) Delete(c echo.Context) error {
	actor := session.UserID(c)
	if err := h.svc.DeleteOrder(c.Request().Context(), c.Param("id"), actor); err != nil {
		return httputil.InternalError(c, err)
	}
	return httputil.OK(c, map[string]string{"message": "deleted"})
}
