// Command ephemeris-service runs the HTTP sidecar exposing Swiss Ephemeris
// calculations. Configuration is environment-only:
//
//	ADDR      listen address (default ":8080")
//	EPHE_PATH directory with Swiss Ephemeris .se1 files (default: none,
//	          the built-in Moshier approximation is used)
//	LOG_LEVEL debug | info | warn | error (default "info")
//
// "ephemeris-service healthcheck" probes the local /healthz and exits 0/1 —
// for container HEALTHCHECK in images without curl/wget.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
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
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}

	log := newLogger(os.Getenv("LOG_LEVEL"))
	if err := run(log); err != nil {
		log.Error("ephemeris-service failed", "error", err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	engine, err := chart.NewEngine(os.Getenv("EPHE_PATH"), log)
	if err != nil {
		return fmt.Errorf("engine init: %w", err)
	}
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
			return fmt.Errorf("shutdown: %w", err)
		}
		log.Info("ephemeris-service stopped")
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func runHealthcheck() int {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: invalid ADDR %q: %v\n", addr, err)
		return 1
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/healthz") //nolint:gosec // self-probe; port comes from our own ADDR

	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: status %s\n", resp.Status)
		return 1
	}
	return 0
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
