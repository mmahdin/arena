package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/app/composition"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/metrics"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
)

func main() {
	if err := run(); err != nil {
		// Never print err.Error(): provider/SQL errors may contain private data.
		slog.Error("api.exited", observability.ErrorFields(err)...)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		// Config validators return literal, secret-free field names only.
		slog.Error("api.configuration.invalid", "reason", err.Error())
		return errors.New("configuration error")
	}
	output, closeLog, err := observability.Output(os.Getenv("LOG_FILE"))
	if err != nil {
		slog.Error("api.logging.unavailable", observability.ErrorFields(err)...)
		return errors.New("cannot open log file")
	}
	defer closeLog()
	// MON-01B: independent sanitized export; never forward ordinary stderr or
	// LOG_FILE records into Loki. The export is opt-in and local-only.
	if cfg.SafeLogEnabled {
		safe, safeErr := observability.NewSafeLogWriter(os.Getenv("THEIA_SAFE_LOG_EXPORT_PATH"), cfg.Environment)
		if safeErr != nil {
			slog.Error("api.safe_log_export.invalid", "reason", "SAFE_LOG_EXPORT_INVALID")
			return errors.New("safe-log export configuration error")
		}
		defer safe.Close()
		output = io.MultiWriter(output, safe)
	}
	logger := observability.NewLogger(output, cfg.LogLevel, cfg.BusinessTimezone)
	slog.SetDefault(logger)
	logger.Info("api.starting", "environment", cfg.Environment, "log_level", cfg.LogLevel.String())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ot, err := telemetry.New(ctx, cfg.OTelOptions)
	if err != nil {
		logger.Error("api.telemetry.invalid", "reason", "OTEL_INITIALIZATION_FAILED")
		return errors.New("telemetry initialization error")
	}
	cfg.OTelRuntime = ot
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := ot.Shutdown(shutdownCtx); err != nil {
			logger.Warn("api.telemetry.export_failed", "reason", "OTLP_UNAVAILABLE")
		}
	}()

	// MON-01A: the metrics endpoint is a completely separate loopback listener.
	// Never mount it on Gin or expose it through a public reverse proxy.
	if cfg.MetricsOptions.Enabled {
		cfg.MetricsRuntime = metrics.New()
		listener, listenErr := net.Listen("tcp", cfg.MetricsOptions.Address)
		if listenErr != nil {
			logger.Error("api.metrics.listen_failed", "reason", "METRICS_LISTENER_UNAVAILABLE")
			return errors.New("private metrics listener unavailable")
		}
		metricsServer := &http.Server{Handler: cfg.MetricsRuntime.Handler(), ReadHeaderTimeout: 3 * time.Second,
			ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 10 * time.Second}
		metricsDone := make(chan struct{})
		go func() {
			defer close(metricsDone)
			if serveErr := metricsServer.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				logger.Error("api.metrics.server_failed", "reason", "METRICS_LISTENER_UNAVAILABLE")
				stop()
			}
		}()
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = metricsServer.Shutdown(shutdownCtx)
			<-metricsDone
		}()
	}

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("api.database.connect_failed", observability.ErrorFields(err)...)
		return errors.New("database connection failed")
	}
	defer db.Close()
	handler, err := composition.NewAPI(cfg, db)
	if err != nil {
		attrs := observability.ErrorFields(err)
		var setup *composition.RuntimeSetupError
		if errors.As(err, &setup) {
			attrs = append(attrs, "setup_step", setup.Step)
		}
		logger.Error("api.composition.failed", attrs...)
		return errors.New("api composition error")
	}

	workers := make(chan error, 1)
	go func() {
		e := handler.Run(ctx)
		workers <- e
		if e != nil && !errors.Is(e, context.Canceled) {
			health := handler.WorkerHealth()
			logger.Error("ai.core.stopped", "scheduler_stage", health.LastFailureStage,
				"scheduler_code", string(health.LastFailureCode), "scheduler_failures", health.ConsecutiveFailures)
			stop()
		}
	}()

	shutdownDone := make(chan struct{})
	srv := &http.Server{
		Addr: cfg.Address, Handler: handler,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if e := srv.Shutdown(shutdownCtx); e != nil {
			logger.Error("api.shutdown.failed", observability.ErrorFields(e)...)
			_ = srv.Close()
		}
		close(shutdownDone)
	}()
	logger.Info("api.listening", "address", cfg.Address)
	serverErr := srv.ListenAndServe()
	stop()
	<-shutdownDone
	workerErr := <-workers
	if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
		logger.Error("api.server.failed", observability.ErrorFields(serverErr)...)
		return errors.New("API server failed")
	}
	if workerErr != nil && !errors.Is(workerErr, context.Canceled) {
		logger.Error("ai.core.failed", observability.ErrorFields(workerErr)...)
		return errors.New("AI Core workers failed")
	}
	logger.Info("api.stopped")
	return nil
}
