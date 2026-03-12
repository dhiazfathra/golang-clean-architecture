package logging

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// New creates a zerolog.Logger configured for the given environment and level.
// In development mode, output uses zerolog.ConsoleWriter for human-readable logs.
// In all other modes, output is JSON to stdout.
func New(env string, level string) zerolog.Logger {
	var w io.Writer = os.Stdout
	if env == "development" {
		w = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}
	lvl := parseLevel(level)
	return zerolog.New(w).With().Timestamp().Logger().Level(lvl)
}

// Noop returns a disabled logger suitable for tests.
func Noop() zerolog.Logger {
	return zerolog.Nop()
}

func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(s) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}
