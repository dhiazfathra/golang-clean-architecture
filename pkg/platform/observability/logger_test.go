package observability

import (
	"bytes"
	"context"
	"testing"

	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestTracedLogger_NoSpan(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	out := TracedLogger(logger, context.Background())
	out.Info().Msg("no span")

	s := buf.String()
	assert.NotContains(t, s, "dd.trace_id", "dd.trace_id should not be present without an active span")
	assert.NotContains(t, s, "dd.span_id", "dd.span_id should not be present without an active span")
}

func TestTracedLogger_WithSpan(t *testing.T) {
	ddtracer.Start(ddtracer.WithLogger(noopLogger{}))
	defer ddtracer.Stop()

	span := ddtracer.StartSpan("test.op")
	ctx := ddtracer.ContextWithSpan(context.Background(), span)
	defer span.Finish()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	out := TracedLogger(logger, ctx)
	out.Info().Msg("with span")

	s := buf.String()
	assert.Contains(t, s, "dd.trace_id", "output should include dd.trace_id")
	assert.Contains(t, s, "dd.span_id", "output should include dd.span_id")
}

// noopLogger satisfies ddtracer.Logger to suppress output during tests.
type noopLogger struct{}

func (noopLogger) Log(msg string) {}
