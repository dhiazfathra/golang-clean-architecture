package rbac

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/httputil"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

// AuditEvent is the shape returned from the events table for audit queries.
type AuditEvent struct {
	ID        int64           `db:"id"          json:"id"`
	EventType string          `db:"event_type"  json:"event_type"`
	Version   int             `db:"version"     json:"version"`
	Data      json.RawMessage `db:"data"        json:"data"`
	Metadata  json.RawMessage `db:"metadata"    json:"metadata"`
	CreatedAt time.Time       `db:"created_at"  json:"created_at"`
}

type Handler struct {
	svc *Service
	db  *sqlx.DB
}

func NewHandler(svc *Service, db *sqlx.DB) *Handler { return &Handler{svc: svc, db: db} }

// POST /admin/roles.
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

// GET /admin/roles.
func (h *Handler) ListRoles(c echo.Context) error {
	roles, err := h.svc.ListRoles(c.Request().Context())
	if err != nil {
		return httputil.InternalError(c, err)
	}
	return httputil.OK(c, FilterResponse(c, roles))
}

// GET /admin/roles/:id.
func (h *Handler) GetRole(c echo.Context) error {
	role, err := h.svc.GetRoleByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httputil.NotFoundOrError(c, err)
	}
	if role == nil {
		return httputil.NotFound(c)
	}
	return httputil.OK(c, FilterResponse(c, role))
}

// DELETE /admin/roles/:id.
func (h *Handler) DeleteRole(c echo.Context) error {
	if err := h.svc.DeleteRole(c.Request().Context(), c.Param("id"), session.UserID(c)); err != nil {
		return httputil.InternalError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// POST /admin/roles/:id/permissions.
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

// DELETE /admin/roles/:id/permissions/:perm  (perm = "module:action").
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

// GET /admin/users/:id/roles.
func (h *Handler) ListUserRoles(c echo.Context) error {
	roleIDs, err := h.svc.GetRolesForUser(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httputil.InternalError(c, err)
	}
	return httputil.OK(c, map[string][]string{"role_ids": roleIDs})
}

// GET /admin/audit/:type/:id — returns all events for a given aggregate.
// Protected by RequirePermission("audit", "read"); super_admin wildcard covers it.
func (h *Handler) GetAuditHistory(c echo.Context) error {
	aggType := c.Param("type")
	aggID := c.Param("id")
	var events []AuditEvent
	err := h.db.SelectContext(c.Request().Context(), &events, `
		SELECT id, event_type, version, data, metadata, created_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version ASC`, aggType, aggID)
	if err != nil {
		return httputil.InternalError(c, err)
	}
	return httputil.OK(c, events)
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
