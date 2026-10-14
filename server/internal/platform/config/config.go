// Package config loads server settings from environment variables.
// Every option is documented in deploy/.env.example (OPS-03). Secrets can also be
// read from a file named by <VAR>_FILE, for Docker and Kubernetes secrets (SEC-09).
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"passwordmanage/server/internal/platform/logging"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"

	SMTPTLSStartTLS = "starttls"
	SMTPTLSImplicit = "tls"
	SMTPTLSNone     = "none"

	defaultHTTPAddr    = ":8080"
	defaultMetricsAddr = ":9090"
	defaultSMTPPort    = 587
	shutdownTimeout    = 15 * time.Second
)

type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      string
}

type Config struct {
	Env             string
	HTTPAddr        string
	MetricsAddr     string // empty disables the metrics listener
	DatabaseURL     string
	LogLevel        string
	LogFormat       string
	ShutdownTimeout time.Duration
	TrustedProxies  []string
	SMTP            SMTP
}

func Load() (Config, error) {
	cfg := Config{
		Env:             getenv("PM_ENV", EnvProduction),
		HTTPAddr:        getenv("PM_HTTP_ADDR", defaultHTTPAddr),
		MetricsAddr:     getenvAllowEmpty("PM_METRICS_ADDR", defaultMetricsAddr),
		LogLevel:        getenv("PM_LOG_LEVEL", "info"),
		LogFormat:       getenv("PM_LOG_FORMAT", logging.FormatJSON),
		ShutdownTimeout: shutdownTimeout,
		TrustedProxies:  splitList(os.Getenv("PM_TRUSTED_PROXIES")),
		SMTP: SMTP{
			Host:     os.Getenv("PM_SMTP_HOST"),
			Username: os.Getenv("PM_SMTP_USERNAME"),
			From:     os.Getenv("PM_SMTP_FROM"),
			TLS:      getenv("PM_SMTP_TLS", SMTPTLSStartTLS),
		},
	}

	var errList []error
	var err error
	if cfg.DatabaseURL, err = secret("PM_DATABASE_URL"); err != nil {
		errList = append(errList, err)
	}
	if cfg.SMTP.Password, err = secret("PM_SMTP_PASSWORD"); err != nil {
		errList = append(errList, err)
	}
	if cfg.SMTP.Port, err = getenvInt("PM_SMTP_PORT", defaultSMTPPort); err != nil {
		errList = append(errList, err)
	}

	errList = append(errList, cfg.validate()...)
	return cfg, errors.Join(errList...)
}

func (c Config) IsDevelopment() bool { return c.Env == EnvDevelopment }

func (c Config) validate() []error {
	var errList []error
	if c.DatabaseURL == "" {
		errList = append(errList, errors.New("PM_DATABASE_URL (or PM_DATABASE_URL_FILE) is required"))
	}
	if c.Env != EnvDevelopment && c.Env != EnvProduction {
		errList = append(errList, fmt.Errorf("PM_ENV must be %s or %s, got %q", EnvDevelopment, EnvProduction, c.Env))
	}
	if c.SMTP.Host == "" {
		errList = append(errList, errors.New("PM_SMTP_HOST is required"))
	}
	if c.SMTP.From == "" {
		errList = append(errList, errors.New("PM_SMTP_FROM is required"))
	}
	switch c.SMTP.TLS {
	case SMTPTLSStartTLS, SMTPTLSImplicit:
	case SMTPTLSNone:
		// Unencrypted SMTP would expose invite emails on the network.
		if !c.IsDevelopment() {
			errList = append(errList, errors.New("PM_SMTP_TLS=none is only allowed when PM_ENV=development"))
		}
	default:
		errList = append(errList, fmt.Errorf("PM_SMTP_TLS must be starttls, tls or none, got %q", c.SMTP.TLS))
	}
	return errList
}

// secret reads KEY, or the file named by KEY_FILE. Setting both is an error.
func secret(key string) (string, error) {
	value, path := os.Getenv(key), os.Getenv(key+"_FILE")
	if path == "" {
		return value, nil
	}
	if value != "" {
		return "", fmt.Errorf("set only one of %s and %s_FILE", key, key)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // the operator chooses this path in their own deployment config
	if err != nil {
		return "", fmt.Errorf("read %s_FILE: %w", key, err)
	}
	return strings.TrimSpace(string(raw)), nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getenvAllowEmpty lets an explicitly empty value disable a feature.
func getenvAllowEmpty(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number, got %q", key, v)
	}
	return n, nil
}

func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
