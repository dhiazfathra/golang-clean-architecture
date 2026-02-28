package observability

import (
	"context"
	"log/slog"

	dderr "github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
)

// ReportError tags the active span with the error and logs it.
// Call from httputil.InternalError or anywhere an unexpected error occurs.
func ReportError(ctx context.Context, err error) {
	span, ok := ddtracer.SpanFromContext(ctx)
	if ok {
		span.SetTag(dderr.Error, true)
		span.SetTag(dderr.ErrorMsg, err.Error())
	}
	slog.ErrorContext(ctx, "internal error", "error", err)
}
