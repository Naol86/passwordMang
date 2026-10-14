package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setValidEnv(t *testing.T) {
	t.Setenv("PM_ENV", EnvDevelopment)
	t.Setenv("PM_DATABASE_URL", "postgres://pm@localhost/pm")
	t.Setenv("PM_SMTP_HOST", "localhost")
	t.Setenv("PM_SMTP_FROM", "vault@example.com")
	t.Setenv("PM_SMTP_TLS", SMTPTLSNone)
}

func TestLoadValid(t *testing.T) {
	setValidEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPAddr != defaultHTTPAddr || cfg.MetricsAddr != defaultMetricsAddr || cfg.SMTP.Port != defaultSMTPPort {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
}

func TestLoadRejectsPlainSMTPInProduction(t *testing.T) {
	setValidEnv(t)
	t.Setenv("PM_ENV", EnvProduction)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "PM_SMTP_TLS=none") {
		t.Fatalf("expected plain SMTP to be rejected in production, got %v", err)
	}
}

func TestLoadReportsAllMissingSettings(t *testing.T) {
	for _, key := range []string{"PM_DATABASE_URL", "PM_SMTP_HOST", "PM_SMTP_FROM"} {
		t.Setenv(key, "")
	}
	_, err := Load()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"PM_DATABASE_URL", "PM_SMTP_HOST", "PM_SMTP_FROM"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %s: %v", want, err)
		}
	}
}

func TestSecretFromFile(t *testing.T) {
	setValidEnv(t)
	path := filepath.Join(t.TempDir(), "db-url")
	if err := os.WriteFile(path, []byte("postgres://from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PM_DATABASE_URL", "")
	t.Setenv("PM_DATABASE_URL_FILE", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DatabaseURL != "postgres://from-file" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestSecretRejectsBothValueAndFile(t *testing.T) {
	setValidEnv(t)
	t.Setenv("PM_DATABASE_URL_FILE", filepath.Join(t.TempDir(), "unused"))
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "set only one of") {
		t.Fatalf("expected conflict error, got %v", err)
	}
}
