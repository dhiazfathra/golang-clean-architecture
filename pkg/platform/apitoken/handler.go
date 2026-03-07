package apitoken

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/httputil"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type createTokenRequest struct {
	Name     string `json:"name"`
	TTLHours int    `json:"ttl_hours"`
}

type createTokenResponse struct {
	Token     string    `json:"token"`
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (h *Handler) Create(c echo.Context) error {
	var req createTokenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	if req.TTLHours < 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ttl_hours must be >= 1"})
	}

	userID := session.UserID(c)
	ttl := time.Duration(req.TTLHours) * time.Hour

	raw, token, err := h.svc.Create(c.Request().Context(), req.Name, userID, ttl)
	if err != nil {
		return httputil.InternalError(c, err)
	}
	return c.JSON(http.StatusCreated, createTokenResponse{
		Token:     raw,
		ID:        token.ID,
		Name:      token.Name,
		ExpiresAt: token.ExpiresAt,
	})
}

func (h *Handler) List(c echo.Context) error {
	userID := session.UserID(c)
	tokens, err := h.svc.List(c.Request().Context(), userID)
	if err != nil {
		return httputil.InternalError(c, err)
	}
	return c.JSON(http.StatusOK, tokens)
}

func (h *Handler) Revoke(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid token id"})
	}
	userID := session.UserID(c)
	if err := h.svc.Revoke(c.Request().Context(), id, userID); err != nil {
		return httputil.InternalError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
