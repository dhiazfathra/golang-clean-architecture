package health

import (
	"context"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/valkey-io/valkey-go"
)

type Handler struct {
	db     *sqlx.DB
	valkey valkey.Client
}

func NewHandler(db *sqlx.DB, vk valkey.Client) *Handler {
	return &Handler{db: db, valkey: vk}
}

// Live handles GET /health — liveness probe.
// Returns 200 immediately; indicates the process is running.
func (h *Handler) Live(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Ready handles GET /health/ready — readiness probe.
// Pings PostgreSQL and Valkey with a 3-second timeout.
// Returns 200 when both are reachable; 503 otherwise.
func (h *Handler) Ready(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{}
	healthy := true

	if err := h.db.PingContext(ctx); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		checks["database"] = "ok"
	}

	if err := h.valkey.Do(ctx, h.valkey.B().Ping().Build()).Error(); err != nil {
		checks["valkey"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		checks["valkey"] = "ok"
	}

	body := map[string]any{"status": "ok", "checks": checks}
	if !healthy {
		body["status"] = "unhealthy"
		return c.JSON(http.StatusServiceUnavailable, body)
	}
	return c.JSON(http.StatusOK, body)
}
