package rbac

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/httputil"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// POST /admin/roles
func (h *Handler) CreateRole(c echo.Context) error {
	var body struct {
		Name        string       `json:"name"`
		Description string       `json:"description"`
		Permissions []Permission `json:"permissions"`
	}
	if err := c.Bind(&body); err != nil {
		return httputil.BadRequest(c, "invalid request body")
	}
	if body.Name == "" {
		return httputil.BadRequest(c, "name is required")
	}
	cmd := CreateRoleCmd{
		Name:        body.Name,
		Description: body.Description,
		Permissions: body.Permissions,
		Actor:       session.UserID(c),
	}
	if err := h.svc.CreateRole(c.Request().Context(), cmd); err != nil {
		return httputil.InternalError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]string{"id": "role_" + body.Name})
}

// GET /admin/roles
func (h *Handler) ListRoles(c echo.Context) error {
	roles, err := h.svc.ListRoles(c.Request().Context())
	if err != nil {
		return httputil.InternalError(c, err)
	}
	return httputil.OK(c, FilterResponse(c, roles))
}

// GET /admin/roles/:id
func (h *Handler) GetRole(c echo.Context) error {
	role, err := h.svc.GetRoleByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httputil.NotFoundOrError(c, err)
	}
	return httputil.OK(c, FilterResponse(c, role))
}

// DELETE /admin/roles/:id
func (h *Handler) DeleteRole(c echo.Context) error {
	if err := h.svc.DeleteRole(c.Request().Context(), c.Param("id"), session.UserID(c)); err != nil {
		return httputil.InternalError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// POST /admin/roles/:id/permissions
func (h *Handler) GrantPermission(c echo.Context) error {
	var perm Permission
	if err := c.Bind(&perm); err != nil {
		return httputil.BadRequest(c, "invalid request body")
	}
	if err := h.svc.GrantPermission(c.Request().Context(), c.Param("id"), perm, session.UserID(c)); err != nil {
		return httputil.InternalError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// DELETE /admin/roles/:id/permissions/:perm  (perm = "module:action")
func (h *Handler) RevokePermission(c echo.Context) error {
	module, action := parsePermParam(c.Param("perm"))
	if module == "" || action == "" {
		return httputil.BadRequest(c, "perm must be module:action")
	}
	if err := h.svc.RevokePermission(c.Request().Context(), c.Param("id"), module, action, session.UserID(c)); err != nil {
		return httputil.InternalError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// GET /admin/users/:id/roles
func (h *Handler) ListUserRoles(c echo.Context) error {
	roleIDs, err := h.svc.GetRolesForUser(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httputil.InternalError(c, err)
	}
	return httputil.OK(c, map[string][]string{"role_ids": roleIDs})
}

// parsePermParam splits "module:action" into its parts.
func parsePermParam(s string) (module, action string) {
	for i, ch := range s {
		if ch == ':' {
			return s[:i], s[i+1:]
		}
	}
	return "", ""
}
