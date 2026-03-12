package observability

import (
	"context"

	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/rs/zerolog"
)

// TracedLogger returns a sub-logger enriched with dd.trace_id and dd.span_id
// from the active Datadog span in ctx. If no span is present, the original
// logger is returned unchanged.
func TracedLogger(logger zerolog.Logger, ctx context.Context) zerolog.Logger {
	span, ok := ddtracer.SpanFromContext(ctx)
	if !ok {
		return logger
	}
	return logger.With().
		Uint64("dd.trace_id", span.Context().TraceIDLower()).
		Uint64("dd.span_id", span.Context().SpanID()).
		Logger()
}
