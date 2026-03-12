package auth

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/httputil"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return httputil.BadRequest(c, "invalid body")
	}
	if err := req.Validate(); err != nil {
		return httputil.BadRequest(c, err.Error())
	}
	meta := map[string]string{
		"user_agent": c.Request().Header.Get("User-Agent"),
		"ip":         c.RealIP(),
	}
	sess, err := h.svc.Login(c.Request().Context(), req, meta)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	c.SetCookie(&http.Cookie{
		Name:     "session_id",
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})
	return httputil.OK(c, map[string]string{"message": "logged in"})
}

func (h *Handler) Logout(c echo.Context) error {
	cookie, err := c.Cookie("session_id")
	if err != nil {
		return httputil.OK(c, map[string]string{"message": "ok"})
	}
	_ = h.svc.Logout(c.Request().Context(), cookie.Value)
	c.SetCookie(&http.Cookie{Name: "session_id", Path: "/", MaxAge: -1})
	return httputil.OK(c, map[string]string{"message": "logged out"})
}

func (h *Handler) Me(c echo.Context) error {
	userID := session.UserID(c)
	return httputil.OK(c, map[string]string{"user_id": userID})
}
