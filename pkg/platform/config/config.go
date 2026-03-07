package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Application
	ListenAddr  string `yaml:"listen_addr"`
	Env         string `yaml:"env"`
	ServiceName string `yaml:"service_name"`

	// Database
	DatabaseURL    string `yaml:"database_url"`
	DBMaxOpenConns int    `yaml:"db_max_open_conns"`
	DBMaxIdleConns int    `yaml:"db_max_idle_conns"`

	// Session
	ValkeyURL  string        `yaml:"valkey_url"`
	SessionTTL time.Duration `yaml:"session_ttl"`

	// Seeder
	SeedSuperAdminPassword    string `yaml:"seed_super_admin_password"`
	SeedDefaultModulePassword string `yaml:"seed_default_module_password"`

	// Observability
	StatsdAddr      string `yaml:"statsd_addr"`
	StatsdNamespace string `yaml:"statsd_namespace"`

	// Feature Flags
	FeatureFlagRefreshTTL time.Duration `yaml:"feature_flag_refresh_ttl"`

	// Env Vars
	EnvVarRefreshTTL time.Duration `yaml:"env_var_refresh_ttl"`

	// API Tokens
	APITokenRefreshTTL time.Duration `yaml:"api_token_refresh_ttl"`
}

func MustLoad() *Config {
	cfg := &Config{
		ListenAddr:            ":8080",
		Env:                   "development",
		ServiceName:           "golang-clean-arch",
		DBMaxOpenConns:        25,
		DBMaxIdleConns:        5,
		SessionTTL:            24 * time.Hour,
		StatsdAddr:            "localhost:8125",
		StatsdNamespace:       "golang_clean_arch.",
		FeatureFlagRefreshTTL: 30 * time.Second,
		EnvVarRefreshTTL:      30 * time.Second,
		APITokenRefreshTTL:    30 * time.Second,
	}

	loadYAML(cfg)
	applyEnvOverrides(cfg)

	if err := cfg.validate(); err != nil {
		panic(fmt.Sprintf("config: %v", err))
	}
	return cfg
}

func loadYAML(cfg *Config) {
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		return
	}

	// Scopes reads to a fixed directory to avoid G703/G304 — path traversal
	root, err := os.OpenRoot(".")
	if err != nil {
		panic(fmt.Sprintf("config: open root: %v", err))
	}
	defer root.Close()

	f, err := root.Open(filepath.Clean(path))
	if err != nil {
		panic(fmt.Sprintf("config: open file: %v", err))
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		panic(fmt.Sprintf("config: read file: %v", err))
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		panic(fmt.Sprintf("config: parse yaml: %v", err))
	}
}

func applyEnvOverrides(cfg *Config) {
	overrideStr("DATABASE_URL", &cfg.DatabaseURL)
	overrideStr("VALKEY_URL", &cfg.ValkeyURL)
	overrideStr("ENV", &cfg.Env)
	overrideStr("LISTEN_ADDR", &cfg.ListenAddr)
	overrideStr("SERVICE_NAME", &cfg.ServiceName)
	overrideStr("STATSD_ADDR", &cfg.StatsdAddr)
	overrideStr("STATSD_NAMESPACE", &cfg.StatsdNamespace)
	overrideStr("SEED_SUPER_ADMIN_PASSWORD", &cfg.SeedSuperAdminPassword)
	overrideStr("SEED_DEFAULT_MODULE_PASSWORD", &cfg.SeedDefaultModulePassword)
	overrideInt("DB_MAX_OPEN_CONNS", &cfg.DBMaxOpenConns)
	overrideInt("DB_MAX_IDLE_CONNS", &cfg.DBMaxIdleConns)
	overrideDuration("SESSION_TTL", &cfg.SessionTTL)
	overrideDuration("FEATURE_FLAG_REFRESH_TTL", &cfg.FeatureFlagRefreshTTL)
	overrideDuration("ENV_VAR_REFRESH_TTL", &cfg.EnvVarRefreshTTL)
	overrideDuration("API_TOKEN_REFRESH_TTL", &cfg.APITokenRefreshTTL)
}

func overrideStr(key string, target *string) {
	if v := os.Getenv(key); v != "" {
		*target = v
	}
}

func overrideInt(key string, target *int) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*target = n
		}
	}
}

func overrideDuration(key string, target *time.Duration) {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			*target = d
		}
	}
}

func (c *Config) validate() error {
	var errs []string
	if c.DatabaseURL == "" {
		errs = append(errs, "DATABASE_URL is required")
	}
	if c.ValkeyURL == "" {
		errs = append(errs, "VALKEY_URL is required")
	}
	validEnvs := map[string]bool{"development": true, "staging": true, "production": true, "test": true}
	if !validEnvs[c.Env] {
		errs = append(errs, fmt.Sprintf("ENV must be development|staging|production|test, got %q", c.Env))
	}
	if c.DBMaxOpenConns < 1 {
		errs = append(errs, "DB_MAX_OPEN_CONNS must be >= 1")
	}
	if c.DBMaxIdleConns < 0 {
		errs = append(errs, "DB_MAX_IDLE_CONNS must be >= 0")
	}
	if c.SessionTTL < time.Minute {
		errs = append(errs, "SESSION_TTL must be >= 1m")
	}
	if c.FeatureFlagRefreshTTL < time.Second {
		errs = append(errs, "FEATURE_FLAG_REFRESH_TTL must be >= 1s")
	}
	if c.EnvVarRefreshTTL < time.Second {
		errs = append(errs, "ENV_VAR_REFRESH_TTL must be >= 1s")
	}
	if c.APITokenRefreshTTL < time.Second {
		errs = append(errs, "API_TOKEN_REFRESH_TTL must be >= 1s")
	}
	if len(errs) > 0 {
		return fmt.Errorf("validation:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func MustGet(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("config: required env var %s is not set", key))
	}
	return v
}

func GetOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
