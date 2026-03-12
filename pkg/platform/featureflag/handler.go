package featureflag

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/httputil"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

type Handler struct {
	svc    *Service
	logger zerolog.Logger
}

func NewHandler(svc *Service, logger zerolog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

type createFlagRequest struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

type toggleRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *Handler) List(c echo.Context) error {
	flags, err := h.svc.List(c.Request().Context())
	if err != nil {
		return httputil.InternalError(c, h.logger, err)
	}
	return c.JSON(http.StatusOK, flags)
}

func (h *Handler) Create(c echo.Context) error {
	var req createFlagRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if req.Key == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key is required"})
	}
	userID := session.UserID(c)
	f, err := h.svc.Create(c.Request().Context(), req.Key, req.Description, req.Enabled, userID)
	if err != nil {
		return httputil.InternalError(c, h.logger, err)
	}
	return c.JSON(http.StatusCreated, f)
}

func (h *Handler) Toggle(c echo.Context) error {
	key := c.Param("key")
	var req toggleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	userID := session.UserID(c)
	if err := h.svc.Toggle(c.Request().Context(), key, req.Enabled, userID); err != nil {
		if err.Error() == "feature flag not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return httputil.InternalError(c, h.logger, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) Delete(c echo.Context) error {
	key := c.Param("key")
	userID := session.UserID(c)
	if err := h.svc.Delete(c.Request().Context(), key, userID); err != nil {
		if err.Error() == "feature flag not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return httputil.InternalError(c, h.logger, err)
	}
	return c.NoContent(http.StatusNoContent)
}
