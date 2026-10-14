// Package testdb gives each test its own freshly migrated PostgreSQL database.
// Set PM_TEST_DATABASE_URL to a server where the user may create databases
// (the dev Compose Postgres works); tests are skipped when it is unset.
package testdb

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"passwordmanage/server/internal/adapter/jobs"
	"passwordmanage/server/internal/adapter/postgres"
	"passwordmanage/server/migrations"
)

const (
	envURL       = "PM_TEST_DATABASE_URL"
	setupTimeout = 2 * time.Minute
	// Serializes template creation across test binaries running in parallel.
	templateLockKey = 7_201_014
)

// New returns a pool connected to a new database cloned from a migrated
// template. The database is dropped when the test ends.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	baseURL := lookupURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), setupTimeout)
	defer cancel()

	admin, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatalf("testdb: connect: %v", err)
	}
	defer func() { _ = admin.Close(context.Background()) }() // test helper connection; nothing to recover

	template, err := ensureTemplate(ctx, admin, baseURL)
	if err != nil {
		t.Fatalf("testdb: template: %v", err)
	}

	name := "pm_test_" + randomSuffix(t)
	if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s",
		pgx.Identifier{name}.Sanitize(), pgx.Identifier{template}.Sanitize())); err != nil {
		t.Fatalf("testdb: create database: %v", err)
	}

	pool, err := postgres.Open(ctx, withDatabase(t, baseURL, name))
	if err != nil {
		t.Fatalf("testdb: open: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		dropCtx, dropCancel := context.WithTimeout(context.Background(), setupTimeout)
		defer dropCancel()
		conn, err := pgx.Connect(dropCtx, baseURL)
		if err != nil {
			t.Errorf("testdb: connect for cleanup: %v", err)
			return
		}
		defer func() { _ = conn.Close(dropCtx) }() // test helper connection; nothing to recover
		if _, err := conn.Exec(dropCtx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", pgx.Identifier{name}.Sanitize())); err != nil {
			t.Errorf("testdb: drop database: %v", err)
		}
	})
	return pool
}

func lookupURL(t testing.TB) string {
	t.Helper()
	baseURL := os.Getenv(envURL)
	if baseURL == "" {
		t.Skipf("set %s to run database tests (see server/Makefile: make test)", envURL)
	}
	return baseURL
}

// ensureTemplate builds a migrated template database named after a hash of the
// migrations, so a schema change automatically produces a new template.
func ensureTemplate(ctx context.Context, admin *pgx.Conn, baseURL string) (string, error) {
	hash, err := migrationsHash()
	if err != nil {
		return "", err
	}
	name := "pm_template_" + hash

	if _, err := admin.Exec(ctx, "SELECT pg_advisory_lock($1)", templateLockKey); err != nil {
		return "", err
	}
	defer func() { _, _ = admin.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", templateLockKey) }()

	var exists bool
	if err := admin.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		return "", err
	}
	if exists {
		return name, nil
	}

	building := name + "_building"
	if _, err := admin.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", pgx.Identifier{building}.Sanitize())); err != nil {
		return "", err
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{building}.Sanitize())); err != nil {
		return "", err
	}
	buildingURL, err := replaceDatabase(baseURL, building)
	if err != nil {
		return "", err
	}
	if err := migrate(ctx, buildingURL); err != nil {
		return "", err
	}
	// Rename only after a full migration, so a crash never leaves a half-built template behind.
	if _, err := admin.Exec(ctx, fmt.Sprintf("ALTER DATABASE %s RENAME TO %s",
		pgx.Identifier{building}.Sanitize(), pgx.Identifier{name}.Sanitize())); err != nil {
		return "", err
	}
	return name, nil
}

func migrate(ctx context.Context, dbURL string) error {
	if err := postgres.Migrate(ctx, dbURL); err != nil {
		return err
	}
	pool, err := postgres.Open(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	return jobs.Migrate(ctx, pool)
}

func migrationsHash() (string, error) {
	names, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return "", err
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		data, err := fs.ReadFile(migrations.FS, n)
		if err != nil {
			return "", err
		}
		h.Write([]byte(n))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

func withDatabase(t testing.TB, baseURL, name string) string {
	t.Helper()
	u, err := replaceDatabase(baseURL, name)
	if err != nil {
		t.Fatalf("testdb: %v", err)
	}
	return u
}

func replaceDatabase(baseURL, name string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("%s must be a postgres:// URL: %w", envURL, err)
	}
	u.Path = "/" + name
	return u.String(), nil
}

func randomSuffix(t testing.TB) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("testdb: random: %v", err)
	}
	return hex.EncodeToString(b)
}
