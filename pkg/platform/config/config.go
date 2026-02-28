package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr                string `yaml:"listen_addr"`
	DatabaseURL               string `yaml:"database_url"`
	ValkeyURL                 string `yaml:"valkey_url"`
	SeedSuperAdminPassword    string `yaml:"seed_super_admin_password"`
	SeedDefaultModulePassword string `yaml:"seed_default_module_password"`
}

func MustLoad() *Config {
	cfg := &Config{
		ListenAddr: ":8080",
	}
	// YAML file optional; env vars override
	if path := os.Getenv("CONFIG_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("config: read file: %v", err))
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			panic(fmt.Sprintf("config: parse yaml: %v", err))
		}
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("VALKEY_URL"); v != "" {
		cfg.ValkeyURL = v
	}
	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		cfg.ListenAddr = v
	}
	if v := os.Getenv("SEED_SUPER_ADMIN_PASSWORD"); v != "" {
		cfg.SeedSuperAdminPassword = v
	}
	if v := os.Getenv("SEED_DEFAULT_MODULE_PASSWORD"); v != "" {
		cfg.SeedDefaultModulePassword = v
	}
	return cfg
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
