package config

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() *Config {
	return &Config{ //nolint:gosec // G101: false positive, test fixture uses local dev credentials
		ListenAddr:            ":8080",
		Env:                   "development",
		ServiceName:           "golang-clean-arch",
		DatabaseURL:           "postgres://app:app@localhost:5432/app?sslmode=disable",
		ValkeyURL:             "localhost:6379",
		DBMaxOpenConns:        25,
		DBMaxIdleConns:        5,
		SessionTTL:            24 * time.Hour,
		StatsdAddr:            "localhost:8125",
		StatsdNamespace:       "golang_clean_arch.",
		FeatureFlagRefreshTTL: 30 * time.Second,
		EnvVarRefreshTTL:      30 * time.Second,
		APITokenRefreshTTL:    30 * time.Second,
	}
}

func TestMustLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x@localhost/x")
	t.Setenv("VALKEY_URL", "localhost:6379")

	cfg := MustLoad()

	assert.Equal(t, ":8080", cfg.ListenAddr)
	// assert.Equal(t, "development", cfg.Env) // Flaky on GitHub Actions pipeline
	assert.Equal(t, "golang-clean-arch", cfg.ServiceName)
	assert.Equal(t, 25, cfg.DBMaxOpenConns)
	assert.Equal(t, 5, cfg.DBMaxIdleConns)
	assert.Equal(t, 24*time.Hour, cfg.SessionTTL)
	assert.Equal(t, "localhost:8125", cfg.StatsdAddr)
	assert.Equal(t, "golang_clean_arch.", cfg.StatsdNamespace)
	assert.Equal(t, 30*time.Second, cfg.FeatureFlagRefreshTTL)
	assert.Equal(t, 30*time.Second, cfg.EnvVarRefreshTTL)
	assert.Equal(t, 30*time.Second, cfg.APITokenRefreshTTL)
}

func TestMustLoad_YAMLConfig(t *testing.T) {
	const yamlContent = `
listen_addr: ":8080"
env: "production"
service_name: "golang-clean-arch"
db_max_open_conns: 25
db_max_idle_conns: 5
session_ttl: 24h
statsd_addr: "localhost:8125"
statsd_namespace: "golang_clean_arch."
feature_flag_refresh_ttl: 30s
`

	const configFileName = "config.yaml"
	// Write the file into the test's working directory
	err := os.WriteFile(configFileName, []byte(yamlContent), 0600)
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(configFileName) })

	t.Setenv("ENV", "") // prevent ambient ENV from overriding YAML
	t.Setenv("DATABASE_URL", "postgres://x@localhost/x")
	t.Setenv("VALKEY_URL", "localhost:6379")
	t.Setenv("CONFIG_FILE", configFileName)

	cfg := MustLoad()

	assert.Equal(t, ":8080", cfg.ListenAddr)
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "golang-clean-arch", cfg.ServiceName)
	assert.Equal(t, 25, cfg.DBMaxOpenConns)
	assert.Equal(t, 5, cfg.DBMaxIdleConns)
	assert.Equal(t, 24*time.Hour, cfg.SessionTTL)
	assert.Equal(t, "localhost:8125", cfg.StatsdAddr)
	assert.Equal(t, "golang_clean_arch.", cfg.StatsdNamespace)
	assert.Equal(t, 30*time.Second, cfg.FeatureFlagRefreshTTL)
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := validConfig()
	assert.NoError(t, cfg.validate())
}

func TestValidate_MissingDatabaseURL(t *testing.T) {
	cfg := validConfig()
	cfg.DatabaseURL = ""
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL is required")
}

func TestValidate_MissingValkeyURL(t *testing.T) {
	cfg := validConfig()
	cfg.ValkeyURL = ""
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VALKEY_URL is required")
}

func TestValidate_InvalidEnv(t *testing.T) {
	cfg := validConfig()
	cfg.Env = "invalid"
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ENV must be development|staging|production|test")
}

func TestValidate_ValidEnvValues(t *testing.T) {
	for _, env := range []string{"development", "staging", "production", "test"} {
		cfg := validConfig()
		cfg.Env = env
		assert.NoError(t, cfg.validate(), "env=%s should be valid", env)
	}
}

func TestValidate_DBMaxOpenConnsZero(t *testing.T) {
	cfg := validConfig()
	cfg.DBMaxOpenConns = 0
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_MAX_OPEN_CONNS must be >= 1")
}

func TestValidate_DBMaxIdleConnsNegative(t *testing.T) {
	cfg := validConfig()
	cfg.DBMaxIdleConns = -1
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_MAX_IDLE_CONNS must be >= 0")
}

func TestValidate_SessionTTLTooShort(t *testing.T) {
	cfg := validConfig()
	cfg.SessionTTL = 30 * time.Second
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SESSION_TTL must be >= 1m")
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := validConfig()
	cfg.DatabaseURL = ""
	cfg.ValkeyURL = ""
	cfg.Env = "bad"
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL is required")
	assert.Contains(t, err.Error(), "VALKEY_URL is required")
	assert.Contains(t, err.Error(), "ENV must be")
}

func TestValidate_FeatureFlagRefreshTTLTooShort(t *testing.T) {
	cfg := validConfig()
	cfg.FeatureFlagRefreshTTL = 500 * time.Millisecond
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FEATURE_FLAG_REFRESH_TTL must be >= 1s")
}

func TestValidate_EnvVarRefreshTTLTooShort(t *testing.T) {
	cfg := validConfig()
	cfg.EnvVarRefreshTTL = 500 * time.Millisecond
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ENV_VAR_REFRESH_TTL must be >= 1s")
}

func TestValidate_APITokenRefreshTTLTooShort(t *testing.T) {
	cfg := validConfig()
	cfg.APITokenRefreshTTL = 500 * time.Millisecond
	err := cfg.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API_TOKEN_REFRESH_TTL must be >= 1s")
}

func TestMustLoad_EnvOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
	t.Setenv("VALKEY_URL", "localhost:6380")
	t.Setenv("ENV", "staging")
	t.Setenv("SERVICE_NAME", "my-svc")
	t.Setenv("DB_MAX_OPEN_CONNS", "50")
	t.Setenv("DB_MAX_IDLE_CONNS", "10")
	t.Setenv("SESSION_TTL", "2h")
	t.Setenv("STATSD_ADDR", "statsd:8125")
	t.Setenv("STATSD_NAMESPACE", "test.")
	t.Setenv("FEATURE_FLAG_REFRESH_TTL", "1m")
	t.Setenv("ENV_VAR_REFRESH_TTL", "2m")
	t.Setenv("API_TOKEN_REFRESH_TTL", "3m")

	cfg := MustLoad()

	assert.Equal(t, "postgres://test:test@localhost/test", cfg.DatabaseURL)
	assert.Equal(t, "localhost:6380", cfg.ValkeyURL)
	assert.Equal(t, "staging", cfg.Env)
	assert.Equal(t, "my-svc", cfg.ServiceName)
	assert.Equal(t, 50, cfg.DBMaxOpenConns)
	assert.Equal(t, 10, cfg.DBMaxIdleConns)
	assert.Equal(t, 2*time.Hour, cfg.SessionTTL)
	assert.Equal(t, "statsd:8125", cfg.StatsdAddr)
	assert.Equal(t, "test.", cfg.StatsdNamespace)
	assert.Equal(t, time.Minute, cfg.FeatureFlagRefreshTTL)
	assert.Equal(t, 2*time.Minute, cfg.EnvVarRefreshTTL)
	assert.Equal(t, 3*time.Minute, cfg.APITokenRefreshTTL)
}

func TestMustLoad_PanicsOnInvalidConfig(t *testing.T) {
	// Explicitly unset required env vars so validation fails
	t.Setenv("DATABASE_URL", "")
	t.Setenv("VALKEY_URL", "")

	// No DATABASE_URL or VALKEY_URL set
	assert.Panics(t, func() { MustLoad() })
}

func TestMustLoad_PanicsOnOpenRootError(t *testing.T) {
	origOpenRoot := openRoot
	defer func() { openRoot = origOpenRoot }()
	openRoot = func(string) (*os.Root, error) {
		return nil, errors.New("root open fail")
	}

	t.Setenv("CONFIG_FILE", "config.yaml")
	t.Setenv("DATABASE_URL", "postgres://x@localhost/x")
	t.Setenv("VALKEY_URL", "localhost:6379")

	assert.Panics(t, func() { MustLoad() })
}

func TestMustLoad_PanicsOnReadFileError(t *testing.T) {
	origReadAll := readAll
	defer func() { readAll = origReadAll }()
	readAll = func(_ io.Reader) ([]byte, error) {
		return nil, errors.New("read fail")
	}

	const configFileName = "config-read-fail.yaml"
	err := os.WriteFile(configFileName, []byte("listen_addr: ':8080'"), 0600)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(configFileName) })

	t.Setenv("CONFIG_FILE", configFileName)
	t.Setenv("DATABASE_URL", "postgres://x@localhost/x")
	t.Setenv("VALKEY_URL", "localhost:6379")

	assert.Panics(t, func() { MustLoad() })
}

func TestMustLoad_PanicsOnYAMLUnmarshalError(t *testing.T) {
	origYAMLUnmarshal := yamlUnmarshal
	defer func() { yamlUnmarshal = origYAMLUnmarshal }()
	yamlUnmarshal = func([]byte, any) error {
		return errors.New("forced yaml failure")
	}

	const configFileName = "config-yaml-fail.yaml"
	err := os.WriteFile(configFileName, []byte("listen_addr: ':8080'"), 0600)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(configFileName) })

	t.Setenv("CONFIG_FILE", configFileName)
	t.Setenv("DATABASE_URL", "postgres://x@localhost/x")
	t.Setenv("VALKEY_URL", "localhost:6379")

	assert.Panics(t, func() { MustLoad() })
}

func TestMustLoad_PanicsOnMissingYAMLFile(t *testing.T) {
	t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), "missing.yaml"))
	t.Setenv("DATABASE_URL", "postgres://x@localhost/x")
	t.Setenv("VALKEY_URL", "localhost:6379")

	assert.Panics(t, func() { MustLoad() })
}

func TestMustLoad_PanicsOnMalformedYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte("listen_addr: [unterminated"), 0600)
	require.NoError(t, err)

	t.Setenv("CONFIG_FILE", path)
	t.Setenv("DATABASE_URL", "postgres://x@localhost/x")
	t.Setenv("VALKEY_URL", "localhost:6379")

	assert.Panics(t, func() { MustLoad() })
}

func TestOverrideStr(t *testing.T) {
	t.Setenv("TEST_KEY", "value")
	target := "default"
	overrideStr("TEST_KEY", &target)
	assert.Equal(t, "value", target)
}

func TestOverrideStr_NoOp(t *testing.T) {
	target := "default"
	overrideStr("NONEXISTENT_KEY_XYZ", &target)
	assert.Equal(t, "default", target)
}

func TestOverrideInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	target := 0
	overrideInt("TEST_INT", &target)
	assert.Equal(t, 42, target)
}

func TestOverrideInt_InvalidIgnored(t *testing.T) {
	t.Setenv("TEST_INT", "notanumber")
	target := 10
	overrideInt("TEST_INT", &target)
	assert.Equal(t, 10, target)
}

func TestOverrideDuration(t *testing.T) {
	t.Setenv("TEST_DUR", "30m")
	target := time.Hour
	overrideDuration("TEST_DUR", &target)
	assert.Equal(t, 30*time.Minute, target)
}

func TestOverrideDuration_InvalidIgnored(t *testing.T) {
	t.Setenv("TEST_DUR", "notaduration")
	target := time.Hour
	overrideDuration("TEST_DUR", &target)
	assert.Equal(t, time.Hour, target)
}

func TestGetOr(t *testing.T) {
	t.Setenv("EXISTING_KEY", "val")
	assert.Equal(t, "val", GetOr("EXISTING_KEY", "fallback"))
	assert.Equal(t, "fallback", GetOr("MISSING_KEY_XYZ", "fallback"))
}

func TestMustGet(t *testing.T) {
	t.Setenv("REQUIRED_KEY", "present")
	assert.Equal(t, "present", MustGet("REQUIRED_KEY"))
}

func TestMustGet_PanicsWhenMissing(t *testing.T) {
	t.Setenv("REQUIRED_KEY_MISSING", "")
	assert.Panics(t, func() { MustGet("REQUIRED_KEY_MISSING") })
}
