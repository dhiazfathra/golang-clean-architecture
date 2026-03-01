package user

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/httputil"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

type Handler struct {
	svc *Service
	db  *sqlx.DB
}

func NewHandler(svc *Service, db *sqlx.DB) *Handler { return &Handler{svc: svc, db: db} }

type createUserReq struct {
	Email    string `json:"email"`
	Password string `json:"password"` //nolint:gosec // G117: false positive, this is a request DTO not a hardcoded secret
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

// GET /admin/users/:id — returns a user even if soft-deleted.
func (h *Handler) AdminGetByID(c echo.Context) error {
	u, err := database.GetIncludingDeleted[UserReadModel](c.Request().Context(), h.db,
		`SELECT * FROM users_read WHERE id = $1`, c.Param("id"))
	if err != nil || u == nil {
		return httputil.NotFound(c)
	}
	return httputil.OK(c, rbac.FilterResponse(c, u))
}
