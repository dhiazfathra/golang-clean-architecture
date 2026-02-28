package observability

import (
	"fmt"
	"time"

	ddecho "github.com/DataDog/dd-trace-go/contrib/labstack/echo.v4/v2"
	"github.com/labstack/echo/v4"
)

// EchoMiddleware returns APM tracing + request metrics + panic recovery.
// Apply to the Echo instance before route registration.
func EchoMiddleware(serviceName string) echo.MiddlewareFunc {
	return ddecho.Middleware(ddecho.WithService(serviceName))
}

// RequestMetrics is a middleware that records HTTP latency/error counters via StatsD.
func RequestMetrics() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			status := c.Response().Status
			dur := time.Since(start).Seconds()
			tags := []string{
				fmt.Sprintf("route:%s", c.Path()),
				fmt.Sprintf("method:%s", c.Request().Method),
				fmt.Sprintf("status:%d", status),
			}
			_ = Count("http.request", 1, tags...)
			_ = Histogram("http.request.duration", dur, tags...)
			if status >= 500 {
				_ = Count("http.error", 1, tags...)
			}
			return err
		}
	}
}
