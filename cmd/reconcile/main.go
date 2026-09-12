package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/cotishq/riftdb/internal/collection"
	"github.com/cotishq/riftdb/internal/db"
	"github.com/cotishq/riftdb/internal/document"
	"github.com/cotishq/riftdb/internal/reconcile"
	"github.com/cotishq/riftdb/internal/storage"
)

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
		logger.Error("object storage unreachable", "error", err)
		os.Exit(1)
	}

	collSvc := collection.NewService(collection.NewRepository(pool))
	docSvc := document.NewService(document.NewRepository(pool))
	rec := reconcile.New(store, docSvc)

	n, err := rec.RunOnce(ctx, collSvc.List)
	if err != nil {
		logger.Error("reconcile failed", "error", err, "inserted", n)
		os.Exit(1)
	}
	logger.Info("reconcile complete", "inserted", n)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
