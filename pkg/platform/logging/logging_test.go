package logging

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestNew_Development(t *testing.T) {
	logger := New("development", "debug")
	assert.Equal(t, zerolog.DebugLevel, logger.GetLevel())
}

func TestNew_Production(t *testing.T) {
	logger := New("production", "info")
	assert.Equal(t, zerolog.InfoLevel, logger.GetLevel())
}

func TestNew_DefaultLevel(t *testing.T) {
	logger := New("production", "")
	assert.Equal(t, zerolog.InfoLevel, logger.GetLevel())
}

func TestNoop(t *testing.T) {
	logger := Noop()
	assert.Equal(t, zerolog.Disabled, logger.GetLevel())
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  zerolog.Level
	}{
		{"trace", zerolog.TraceLevel},
		{"TRACE", zerolog.TraceLevel},
		{"debug", zerolog.DebugLevel},
		{"DEBUG", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"INFO", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"WARN", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"ERROR", zerolog.ErrorLevel},
		{"fatal", zerolog.FatalLevel},
		{"FATAL", zerolog.FatalLevel},
		{"unknown", zerolog.InfoLevel},
		{"", zerolog.InfoLevel},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
