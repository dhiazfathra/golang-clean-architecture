package user

import (
	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/httputil"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type createUserReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Create(c echo.Context) error {
	var req createUserReq
	if err := c.Bind(&req); err != nil {
		return httputil.BadRequest(c, "invalid body")
	}
	actor := session.UserID(c)
	id, err := h.svc.CreateUser(c.Request().Context(), CreateUserCmd{
		Email: req.Email, Password: req.Password, Actor: actor,
	})
	if err != nil {
		return httputil.InternalError(c, err)
	}
	return httputil.Created(c, map[string]string{"id": id})
}

func (h *Handler) GetByID(c echo.Context) error {
	u, err := h.svc.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil || u == nil {
		return httputil.NotFound(c)
	}
	return httputil.OK(c, rbac.FilterResponse(c, u))
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
	if err := h.svc.DeleteUser(c.Request().Context(), c.Param("id"), actor); err != nil {
		return httputil.InternalError(c, err)
	}
	return httputil.OK(c, map[string]string{"message": "deleted"})
}
