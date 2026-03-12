package envvar

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
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

type createEnvRequest struct {
	Platform string `json:"platform"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

type updateEnvRequest struct {
	Value string `json:"value"`
}

type envResponse struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Data          any    `json:"data"`
}

type envDataResponse struct {
	ID       string `json:"id"`
	Platform string `json:"platform"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

type paginatedEnvResponse struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Data          any    `json:"data"`
	Meta          any    `json:"meta"`
}

type paginationMeta struct {
	Pagination paginationInfo `json:"pagination"`
}

type paginationInfo struct {
	Page      int `json:"page"`
	PageSize  int `json:"page_size"`
	PageCount int `json:"page_count"`
	Total     int `json:"total"`
}

func toEnvData(e *EnvVar) envDataResponse {
	return envDataResponse{
		ID:       "env:" + e.Platform + ":" + e.Key,
		Platform: e.Platform,
		Key:      e.Key,
		Value:    e.Value,
	}
}

func (h *Handler) CreateEnv(c echo.Context) error {
	var req createEnvRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, envResponse{
			StatusCode: http.StatusBadRequest, StatusMessage: "Bad Request",
			Data: map[string]string{"error": "invalid request body"},
		})
	}
	if req.Platform == "" || req.Key == "" || req.Value == "" {
		return c.JSON(http.StatusBadRequest, envResponse{
			StatusCode: http.StatusBadRequest, StatusMessage: "Bad Request",
			Data: map[string]string{"error": "platform, key, and value are required"},
		})
	}
	if len(req.Platform) > 30 {
		return c.JSON(http.StatusBadRequest, envResponse{
			StatusCode: http.StatusBadRequest, StatusMessage: "Bad Request",
			Data: map[string]string{"error": "platform must be at most 30 characters"},
		})
	}
	if len(req.Key) > 50 {
		return c.JSON(http.StatusBadRequest, envResponse{
			StatusCode: http.StatusBadRequest, StatusMessage: "Bad Request",
			Data: map[string]string{"error": "key must be at most 50 characters"},
		})
	}
	userID := session.UserID(c)
	e, err := h.svc.Create(c.Request().Context(), req.Platform, req.Key, req.Value, userID)
	if err != nil {
		return httputil.InternalError(c, h.logger, err)
	}
	return c.JSON(http.StatusCreated, envResponse{
		StatusCode: http.StatusCreated, StatusMessage: "Created",
		Data: toEnvData(e),
	})
}

func (h *Handler) GetEnv(c echo.Context) error {
	platform := c.Param("platform")
	key := c.Param("key")
	e, err := h.svc.Get(c.Request().Context(), platform, key)
	if err != nil {
		if err.Error() == "env var not found" {
			return c.JSON(http.StatusNotFound, envResponse{
				StatusCode: http.StatusNotFound, StatusMessage: "Not Found",
				Data: map[string]string{"error": err.Error()},
			})
		}
		return httputil.InternalError(c, h.logger, err)
	}
	return c.JSON(http.StatusOK, envResponse{
		StatusCode: http.StatusOK, StatusMessage: "OK",
		Data: toEnvData(e),
	})
}

func (h *Handler) GetEnvsByPlatform(c echo.Context) error {
	platform := c.Param("platform")
	var req database.PageRequest
	if err := c.Bind(&req); err != nil {
		req = database.PageRequest{}
	}
	req.Normalise("key")
	page, err := h.svc.ListByPlatform(c.Request().Context(), platform, req)
	if err != nil {
		return httputil.InternalError(c, h.logger, err)
	}
	data := make([]envDataResponse, len(page.Items))
	for i, e := range page.Items {
		data[i] = toEnvData(&e)
	}
	return c.JSON(http.StatusOK, paginatedEnvResponse{
		StatusCode: http.StatusOK, StatusMessage: "OK",
		Data: data,
		Meta: paginationMeta{
			Pagination: paginationInfo{
				Page:      page.Page,
				PageSize:  page.PageSize,
				PageCount: page.TotalPages,
				Total:     int(page.Total),
			},
		},
	})
}

func (h *Handler) UpdateEnv(c echo.Context) error {
	platform := c.Param("platform")
	key := c.Param("key")
	var req updateEnvRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, envResponse{
			StatusCode: http.StatusBadRequest, StatusMessage: "Bad Request",
			Data: map[string]string{"error": "invalid request body"},
		})
	}
	if req.Value == "" {
		return c.JSON(http.StatusBadRequest, envResponse{
			StatusCode: http.StatusBadRequest, StatusMessage: "Bad Request",
			Data: map[string]string{"error": "value is required"},
		})
	}
	userID := session.UserID(c)
	e, err := h.svc.Update(c.Request().Context(), platform, key, req.Value, userID)
	if err != nil {
		if err.Error() == "env var not found" {
			return c.JSON(http.StatusNotFound, envResponse{
				StatusCode: http.StatusNotFound, StatusMessage: "Not Found",
				Data: map[string]string{"error": err.Error()},
			})
		}
		return httputil.InternalError(c, h.logger, err)
	}
	return c.JSON(http.StatusOK, envResponse{
		StatusCode: http.StatusOK, StatusMessage: "OK",
		Data: toEnvData(e),
	})
}

func (h *Handler) DeleteEnv(c echo.Context) error {
	platform := c.Param("platform")
	key := c.Param("key")
	userID := session.UserID(c)
	if err := h.svc.Delete(c.Request().Context(), platform, key, userID); err != nil {
		if err.Error() == "env var not found" {
			return c.JSON(http.StatusNotFound, envResponse{
				StatusCode: http.StatusNotFound, StatusMessage: "Not Found",
				Data: map[string]string{"error": err.Error()},
			})
		}
		return httputil.InternalError(c, h.logger, err)
	}
	return c.JSON(http.StatusOK, envResponse{
		StatusCode: http.StatusOK, StatusMessage: "OK",
		Data: map[string]string{"message": "deleted"},
	})
}
