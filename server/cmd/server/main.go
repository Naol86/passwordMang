// Command server is the composition root: it is the only place that knows the
// concrete adapters and wires them into use cases.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"golang.org/x/sync/errgroup"

	"passwordmanage/server/internal/adapter/httpapi"
	"passwordmanage/server/internal/adapter/jobs"
	"passwordmanage/server/internal/adapter/mail"
	"passwordmanage/server/internal/adapter/postgres"
	"passwordmanage/server/internal/platform/config"
	"passwordmanage/server/internal/platform/logging"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log, err := logging.New(os.Stdout, cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		return err
	}
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := postgres.Migrate(ctx, cfg.DatabaseURL); err != nil {
		return err
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := jobs.Migrate(ctx, pool); err != nil {
		return err
	}

	// River logs every internal step at debug level; keep it at info even when we debug.
	queue, err := jobs.New(pool, mail.NewSMTP(cfg.SMTP), logging.AtLeast(log, slog.LevelInfo).With("component", "jobs"))
	if err != nil {
		return err
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		postgres.NewPoolCollector(pool),
	)

	router, err := httpapi.NewRouter(httpapi.Deps{
		Log:            log,
		DB:             pool,
		Version:        version,
		TrustedProxies: cfg.TrustedProxies,
		Metrics:        registry,
	})
	if err != nil {
		return err
	}

	servers := []*http.Server{newServer(cfg.HTTPAddr, router)}
	if cfg.MetricsAddr != "" {
		servers = append(servers, newServer(cfg.MetricsAddr, httpapi.MetricsHandler(registry)))
	}

	if err := queue.Start(ctx); err != nil {
		return err
	}

	group, groupCtx := errgroup.WithContext(ctx)
	for _, srv := range servers {
		group.Go(func() error {
			log.Info("listening", "addr", srv.Addr, "version", version)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		})
	}
	group.Go(func() error {
		<-groupCtx.Done()
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		var errList []error
		for _, srv := range servers {
			errList = append(errList, srv.Shutdown(shutdownCtx))
		}
		errList = append(errList, queue.Stop(shutdownCtx))
		return errors.Join(errList...)
	})
	return group.Wait()
}

func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}
