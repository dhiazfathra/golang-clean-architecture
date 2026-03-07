package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DataDog/datadog-go/v5/statsd"
	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	ddprofiler "github.com/DataDog/dd-trace-go/v2/profiler"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubStatsdClient struct{ statsd.NoOpClient }

func TestInitNoop(t *testing.T) {
	InitNoop()
	assert.NotNil(t, statsdClient)
	_, ok := statsdClient.(*statsd.NoOpClient)
	assert.True(t, ok)
}

func TestInit(t *testing.T) {
	// Init with default config should not panic
	Init(InitConfig{
		ServiceName:     "test-svc",
		Env:             "test",
		StatsdAddr:      "localhost:8125",
		StatsdNamespace: "test.",
	})
	defer Stop()
}

func TestInit_StatsdFallbackOnError(t *testing.T) {
	origTracerStart := tracerStart
	origProfilerStart := profilerStart
	origNewStatsdClient := newStatsdClient
	origStatsdClient := statsdClient
	defer func() {
		tracerStart = origTracerStart
		profilerStart = origProfilerStart
		newStatsdClient = origNewStatsdClient
		statsdClient = origStatsdClient
	}()

	tracerCalled := false
	profilerCalled := false
	tracerStart = func(opts ...ddtracer.StartOption) error {
		tracerCalled = true
		return nil
	}
	profilerStart = func(opts ...ddprofiler.Option) error {
		profilerCalled = true
		return nil
	}
	newStatsdClient = func(addr string, opts ...statsd.Option) (statsd.ClientInterface, error) {
		assert.Equal(t, "bad:8125", addr)
		return nil, errors.New("statsd down")
	}

	Init(InitConfig{ServiceName: "svc", Env: "test", StatsdAddr: "bad:8125", StatsdNamespace: "test."})

	assert.True(t, tracerCalled)
	assert.True(t, profilerCalled)
	_, ok := statsdClient.(*statsd.NoOpClient)
	assert.True(t, ok)
	Stop()
}

func TestInit_TracerAndProfilerFailuresDoNotPreventStatsd(t *testing.T) {
	origTracerStart := tracerStart
	origProfilerStart := profilerStart
	origNewStatsdClient := newStatsdClient
	origStatsdClient := statsdClient
	defer func() {
		tracerStart = origTracerStart
		profilerStart = origProfilerStart
		newStatsdClient = origNewStatsdClient
		statsdClient = origStatsdClient
	}()

	tracerStart = func(opts ...ddtracer.StartOption) error {
		return errors.New("tracer failed")
	}
	profilerStart = func(opts ...ddprofiler.Option) error {
		return errors.New("profiler failed")
	}
	expectedClient := &stubStatsdClient{}
	newStatsdClient = func(addr string, opts ...statsd.Option) (statsd.ClientInterface, error) {
		assert.Equal(t, "localhost:8125", addr)
		return expectedClient, nil
	}

	Init(InitConfig{ServiceName: "svc", Env: "test", StatsdAddr: "localhost:8125", StatsdNamespace: "test."})

	assert.Same(t, expectedClient, statsdClient)
	Stop()
}

func TestStop(t *testing.T) {
	InitNoop()
	Stop() // should not panic
}

func TestStopNilClient(t *testing.T) {
	old := statsdClient
	statsdClient = nil
	Stop() // should not panic with nil client
	statsdClient = old
}

func TestEchoMiddleware(t *testing.T) {
	mw := EchoMiddleware("test-service")
	assert.NotNil(t, mw)
}

func TestRequestMetrics(t *testing.T) {
	InitNoop()
	e := echo.New()
	mw := RequestMetrics()
	require.NotNil(t, mw)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/test")

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequestMetrics500(t *testing.T) {
	InitNoop()
	e := echo.New()
	mw := RequestMetrics()

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/err")

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusInternalServerError, "fail")
	})
	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestReportError_NoSpan(t *testing.T) {
	InitNoop()
	// No span in context — should not panic
	ReportError(context.Background(), errors.New("test error"))
}

func TestReportError_WithSpan(t *testing.T) {
	ddtracer.Start(ddtracer.WithLogger(noopLogger{}))
	defer ddtracer.Stop()

	span := ddtracer.StartSpan("test.op")
	ctx := ddtracer.ContextWithSpan(context.Background(), span)
	defer span.Finish()

	ReportError(ctx, errors.New("test error"))
}

func TestNewHTTPClient(t *testing.T) {
	InitNoop()
	client := NewHTTPClient()
	assert.NotNil(t, client)
}

func TestCount(t *testing.T) {
	InitNoop()
	err := Count("test.count", 1, "tag:value")
	assert.NoError(t, err)
}

func TestCountNilClient(t *testing.T) {
	old := statsdClient
	statsdClient = nil
	err := Count("test.count", 1)
	assert.NoError(t, err)
	statsdClient = old
}

func TestHistogram(t *testing.T) {
	InitNoop()
	err := Histogram("test.hist", 1.5, "tag:value")
	assert.NoError(t, err)
}

func TestHistogramNilClient(t *testing.T) {
	old := statsdClient
	statsdClient = nil
	err := Histogram("test.hist", 1.5)
	assert.NoError(t, err)
	statsdClient = old
}

func TestGauge(t *testing.T) {
	InitNoop()
	err := Gauge("test.gauge", 42.0, "tag:value")
	assert.NoError(t, err)
}

func TestGaugeNilClient(t *testing.T) {
	old := statsdClient
	statsdClient = nil
	err := Gauge("test.gauge", 42.0)
	assert.NoError(t, err)
	statsdClient = old
}
