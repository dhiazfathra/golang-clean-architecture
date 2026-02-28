package observability

import (
	"context"
	"log/slog"

	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
)

// TracedHandler wraps an slog.Handler to inject dd.trace_id and dd.span_id
// into every log record that has an active Datadog span in its context.
type TracedHandler struct {
	inner slog.Handler
}

func NewTracedHandler(inner slog.Handler) *TracedHandler {
	return &TracedHandler{inner: inner}
}

func (h *TracedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *TracedHandler) Handle(ctx context.Context, r slog.Record) error {
	span, ok := ddtracer.SpanFromContext(ctx)
	if ok {
		r.AddAttrs(
			slog.Uint64("dd.trace_id", span.Context().TraceIDLower()),
			slog.Uint64("dd.span_id", span.Context().SpanID()),
		)
	}
	return h.inner.Handle(ctx, r)
}

func (h *TracedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TracedHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *TracedHandler) WithGroup(name string) slog.Handler {
	return &TracedHandler{inner: h.inner.WithGroup(name)}
}
