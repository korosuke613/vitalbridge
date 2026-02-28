package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/korosuke613/vitalbridge/config"
	"github.com/korosuke613/vitalbridge/handlers"
	"github.com/korosuke613/vitalbridge/middleware"
	"github.com/korosuke613/vitalbridge/store"
)

const Version = "0.1.0"

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Health Ingest Service v%s\n", Version)
		return
	}

	// Bootstrap logger with JSON/stdout defaults (before config is available)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Re-initialize logger with configured level and format
	initLogger(&cfg.Log)

	slog.Info("starting service", "version", Version)

	// Initialize metrics store
	ms := store.NewMetricsStore()

	// Set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ingest", middleware.BearerAuth(cfg.Auth.APIKey, handlers.NewIngestHandler(ms)))
	mux.HandleFunc("/api/health", handlers.NewHealthHandler())
	mux.HandleFunc("/metrics", handlers.NewMetricsHandler(ms))
	mux.HandleFunc("/api/status", middleware.BearerAuth(cfg.Auth.APIKey, handlers.NewStatusHandler(ms)))

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	// TTL cleanup goroutine
	ttl := time.Duration(cfg.Metrics.TTLHours) * time.Hour
	cleanupInterval := time.Duration(cfg.Metrics.CleanupIntervalMinutes) * time.Minute
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			ms.CleanExpired(ttl)
			slog.Debug("expired metrics cleanup completed")
		}
	}()

	// Start server
	go func() {
		slog.Info("listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("service started")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	slog.Info("received signal, shutting down", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}

	slog.Info("service stopped")
}

func initLogger(logCfg *config.LogConfig) {
	level := logCfg.SlogLevel()
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch strings.ToLower(logCfg.Format) {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
