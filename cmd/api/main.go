package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/cotishq/riftdb/internal/api"
	"github.com/cotishq/riftdb/internal/collection"
	"github.com/cotishq/riftdb/internal/db"
	"github.com/cotishq/riftdb/internal/document"
	"github.com/cotishq/riftdb/internal/reconcile"
	"github.com/cotishq/riftdb/internal/storage"
)

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(getenv("LOG_LEVEL", "info")),
	}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, getenv("DATABASE_URL", "postgres://riftdb:riftdb@localhost:5432/riftdb?sslmode=disable"))
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	store, err := storage.NewS3FromEnv(ctx)
	if err != nil {
		logger.Error("failed to configure object storage", "error", err)
		os.Exit(1)
	}
	if err := store.Ping(ctx); err != nil {
		logger.Error("object storage unreachable", "error", err, "bucket", store.Bucket(), "endpoint", store.Endpoint())
		os.Exit(1)
	}
	logger.Info("object storage ready", "bucket", store.Bucket(), "endpoint", store.Endpoint())

	collRepo := collection.NewRepository(pool)
	collSvc := collection.NewService(collRepo)
	collHandler := collection.NewHandler(collSvc)

	docRepo := document.NewRepository(pool)
	docSvc := document.NewService(docRepo)
	docHandler := document.NewHandler(docSvc)

	rec := reconcile.New(store, docSvc)
	interval := parseInterval(getenv("RECONCILE_INTERVAL", "30s"))
	go rec.Loop(ctx, interval, collSvc.List)
	logger.Info("reconciler started", "interval", interval)

	router := api.NewRouter(pool, collHandler, docHandler)

	addr := ":" + getenv("PORT", "8080")
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "addr", addr, "error", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("api server listening", "addr", addr)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseInterval(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 30 * time.Second
	}
	return d
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
