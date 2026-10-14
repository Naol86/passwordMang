// Package config loads server settings from environment variables.
// Every option is documented in deploy/.env.example (OPS-03).
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Env             string // "development" or "production"
	HTTPAddr        string
	DatabaseURL     string
	LogLevel        string
	ShutdownTimeout time.Duration
	TrustedProxies  []string
}

func Load() (Config, error) {
	cfg := Config{
		Env:             getenv("PM_ENV", "production"),
		HTTPAddr:        getenv("PM_HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("PM_DATABASE_URL"),
		LogLevel:        getenv("PM_LOG_LEVEL", "info"),
		ShutdownTimeout: 15 * time.Second,
	}

	if v := os.Getenv("PM_TRUSTED_PROXIES"); v != "" {
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				cfg.TrustedProxies = append(cfg.TrustedProxies, p)
			}
		}
	}

	if cfg.DatabaseURL == "" {
		return cfg, errors.New("PM_DATABASE_URL is required")
	}
	if cfg.Env != "development" && cfg.Env != "production" {
		return cfg, fmt.Errorf("PM_ENV must be development or production, got %q", cfg.Env)
	}
	return cfg, nil
}

func (c Config) IsDevelopment() bool { return c.Env == "development" }

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
