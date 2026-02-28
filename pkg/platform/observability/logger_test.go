package observability

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureHandler captures the last slog.Record passed to Handle.
type captureHandler struct {
	enabled bool
	last    *slog.Record
}

func (c *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return c.enabled }
func (c *captureHandler) Handle(_ context.Context, r slog.Record) error {
	cp := r.Clone()
	c.last = &cp
	return nil
}
func (c *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &captureHandler{enabled: c.enabled}
}
func (c *captureHandler) WithGroup(name string) slog.Handler {
	return &captureHandler{enabled: c.enabled}
}

func TestNewTracedHandler(t *testing.T) {
	inner := &captureHandler{enabled: true}
	h := NewTracedHandler(inner)
	assert.NotNil(t, h)
}

func TestTracedHandler_Enabled(t *testing.T) {
	h := NewTracedHandler(&captureHandler{enabled: true})
	assert.True(t, h.Enabled(context.Background(), slog.LevelInfo))

	h2 := NewTracedHandler(&captureHandler{enabled: false})
	assert.False(t, h2.Enabled(context.Background(), slog.LevelInfo))
}

func TestTracedHandler_WithAttrs(t *testing.T) {
	inner := &captureHandler{enabled: true}
	h := NewTracedHandler(inner)
	h2 := h.WithAttrs([]slog.Attr{slog.String("k", "v")})
	_, ok := h2.(*TracedHandler)
	assert.True(t, ok, "WithAttrs should return *TracedHandler")
}

func TestTracedHandler_WithGroup(t *testing.T) {
	inner := &captureHandler{enabled: true}
	h := NewTracedHandler(inner)
	h2 := h.WithGroup("grp")
	_, ok := h2.(*TracedHandler)
	assert.True(t, ok, "WithGroup should return *TracedHandler")
}

func TestTracedHandler_Handle_NoSpan(t *testing.T) {
	inner := &captureHandler{enabled: true}
	h := NewTracedHandler(inner)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "no span", 0)
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	require.NotNil(t, inner.last)
	var hasTraceID, hasSpanID bool
	inner.last.Attrs(func(a slog.Attr) bool {
		if a.Key == "dd.trace_id" {
			hasTraceID = true
		}
		if a.Key == "dd.span_id" {
			hasSpanID = true
		}
		return true
	})
	assert.False(t, hasTraceID, "dd.trace_id should not be present without an active span")
	assert.False(t, hasSpanID, "dd.span_id should not be present without an active span")
}

func TestTracedHandler_Handle_WithSpan(t *testing.T) {
	ddtracer.Start(ddtracer.WithLogger(noopLogger{}))
	defer ddtracer.Stop()

	span := ddtracer.StartSpan("test.op")
	ctx := ddtracer.ContextWithSpan(context.Background(), span)
	defer span.Finish()

	var buf bytes.Buffer
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := NewTracedHandler(jsonHandler)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "with span", 0)
	err := h.Handle(ctx, r)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "dd.trace_id", "JSON output should include dd.trace_id")
	assert.Contains(t, out, "dd.span_id", "JSON output should include dd.span_id")
}

// noopLogger satisfies ddtracer.Logger to suppress output during tests.
type noopLogger struct{}

func (noopLogger) Log(msg string) {}
