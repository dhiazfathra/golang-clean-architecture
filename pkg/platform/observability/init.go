package observability

import (
	"log/slog"
	"os"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	ddprofiler "github.com/DataDog/dd-trace-go/v2/profiler"
)

var statsdClient statsd.ClientInterface

// InitConfig holds observability initialisation parameters.
type InitConfig struct {
	ServiceName     string
	Env             string
	StatsdAddr      string
	StatsdNamespace string
}

// Init starts the Datadog tracer, continuous profiler, StatsD, and runtime metrics.
// Call once at the very top of main(), before any other initialisation.
func Init(cfg InitConfig) {
	if err := ddtracer.Start(
		ddtracer.WithService(cfg.ServiceName),
		ddtracer.WithEnv(cfg.Env),
		ddtracer.WithRuntimeMetrics(),
	); err != nil {
		slog.Warn("observability: tracer start failed", "error", err)
	}

	if err := ddprofiler.Start(
		ddprofiler.WithService(cfg.ServiceName),
		ddprofiler.WithEnv(cfg.Env),
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
	statsdClient, err = statsd.New(cfg.StatsdAddr,
		statsd.WithNamespace(cfg.StatsdNamespace),
		statsd.WithTags([]string{"service:" + cfg.ServiceName, "env:" + cfg.Env}),
	)
	if err != nil {
		slog.Warn("observability: statsd init failed", "error", err)
		statsdClient = &statsd.NoOpClient{}
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(NewTracedHandler(jsonHandler)))
}

// Stop flushes and stops the tracer and profiler. Call via defer in main().
func Stop() {
	ddtracer.Stop()
	ddprofiler.Stop()
	if statsdClient != nil {
		_ = statsdClient.Close()
	}
}
