package httputil

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

func OK(c echo.Context, data any) error      { return c.JSON(http.StatusOK, data) }
func Created(c echo.Context, data any) error { return c.JSON(http.StatusCreated, data) }
func BadRequest(c echo.Context, msg string) error {
	return c.JSON(http.StatusBadRequest, map[string]string{"error": msg})
}
func InternalError(c echo.Context, err error) error {
	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}
func NotFound(c echo.Context) error {
	return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
}
func NotFoundOrError(c echo.Context, err error) error {
	if errors.Is(err, ErrNotFound) {
		return NotFound(c)
	}
	return InternalError(c, err)
}

var ErrNotFound = errors.New("not found")

func BindAndValidate(c echo.Context, v any) error {
	if err := c.Bind(v); err != nil {
		return BadRequest(c, "invalid request body")
	}
	if err := c.Validate(v); err != nil {
		return BadRequest(c, err.Error())
	}
	return nil
}
