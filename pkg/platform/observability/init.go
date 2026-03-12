package observability

import (
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	ddprofiler "github.com/DataDog/dd-trace-go/v2/profiler"
	"github.com/rs/zerolog"
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
func Init(cfg InitConfig, logger zerolog.Logger) {
	if err := ddtracer.Start(
		ddtracer.WithService(cfg.ServiceName),
		ddtracer.WithEnv(cfg.Env),
		ddtracer.WithRuntimeMetrics(),
	); err != nil {
		logger.Warn().Err(err).Msg("observability: tracer start failed")
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
		logger.Warn().Err(err).Msg("observability: profiler start failed")
	}

	var err error
	statsdClient, err = statsd.New(cfg.StatsdAddr,
		statsd.WithNamespace(cfg.StatsdNamespace),
		statsd.WithTags([]string{"service:" + cfg.ServiceName, "env:" + cfg.Env}),
	)
	if err != nil {
		logger.Warn().Err(err).Msg("observability: statsd init failed")
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
