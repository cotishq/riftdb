package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/cotishq/riftdb/internal/storage"
)

func main() {
	_ = godotenv.Load()

	prefix := "collections/papers"
	if len(os.Args) > 1 {
		prefix = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	store, err := storage.NewS3FromEnv(ctx)
	if err != nil {
		slog.Error("storage config", "error", err)
		os.Exit(1)
	}

	objects, err := store.List(ctx, prefix)
	if err != nil {
		slog.Error("list failed", "error", err, "prefix", prefix, "bucket", store.Bucket())
		os.Exit(1)
	}

	slog.Info("listed objects",
		"count", len(objects),
		"prefix", prefix,
		"bucket", store.Bucket(),
		"endpoint", store.Endpoint(),
	)
	for _, obj := range objects {
		fmt.Println(obj.Key)
	}
}
