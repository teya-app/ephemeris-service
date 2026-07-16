// Command ephemeris-service runs the HTTP sidecar exposing Swiss Ephemeris
// calculations. Configuration is environment-only:
//
//	ADDR      listen address (default ":8080")
//	EPHE_PATH directory with Swiss Ephemeris .se1 files (default: none,
//	          the built-in Moshier approximation is used)
//	LOG_LEVEL debug | info | warn | error (default "info")
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

	"github.com/teya-app/ephemeris-service/internal/chart"
	"github.com/teya-app/ephemeris-service/internal/server"
	"github.com/teya-app/ephemeris-service/internal/sweph"
)

func main() {
	log := newLogger(os.Getenv("LOG_LEVEL"))

	engine := chart.NewEngine(os.Getenv("EPHE_PATH"), log)
	defer sweph.Close()

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.New(engine, log),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	log.Info("ephemeris-service started",
		"addr", addr,
		"engine_version", engine.EngineVersion(),
		"ephemeris", engine.Ephemeris(),
	)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("shutdown failed", "error", err.Error())
			os.Exit(1)
		}
		log.Info("ephemeris-service stopped")
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err.Error())
			os.Exit(1)
		}
	}
}

func newLogger(level string) *slog.Logger {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}
