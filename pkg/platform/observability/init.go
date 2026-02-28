package observability

import (
	"log/slog"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	ddprofiler "github.com/DataDog/dd-trace-go/v2/profiler"
	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
)

var statsdClient statsd.ClientInterface

// Init starts the Datadog tracer, continuous profiler, StatsD, and runtime metrics.
// Call once at the very top of main(), before any other initialisation.
// serviceName should match the Datadog service name (e.g. "golang-clean-arch").
func Init(serviceName, env string) {
	if err := ddtracer.Start(
		ddtracer.WithService(serviceName),
		ddtracer.WithEnv(env),
		ddtracer.WithRuntimeMetrics(),
	); err != nil {
		slog.Warn("observability: tracer start failed", "error", err)
	}

	if err := ddprofiler.Start(
		ddprofiler.WithService(serviceName),
		ddprofiler.WithEnv(env),
		ddprofiler.CPUDuration(60*time.Second),
		ddprofiler.WithProfileTypes(
			ddprofiler.CPUProfile,
			ddprofiler.HeapProfile,
			ddprofiler.GoroutineProfile,
		),
	); err != nil {
		slog.Warn("observability: profiler start failed", "error", err)
	}

	var err error
	statsdClient, err = statsd.New("localhost:8125",
		statsd.WithNamespace("myapp."),
		statsd.WithTags([]string{"service:" + serviceName, "env:" + env}),
	)
	if err != nil {
		slog.Warn("observability: statsd init failed", "error", err)
		statsdClient = &statsd.NoOpClient{}
	}
}

// Stop flushes and stops the tracer and profiler. Call via defer in main().
func Stop() {
	ddtracer.Stop()
	ddprofiler.Stop()
	if statsdClient != nil {
		_ = statsdClient.Close()
	}
}
