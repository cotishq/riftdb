package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/cotishq/riftdb/internal/worker"
)

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	w := worker.New(addr)
	go func() {
		ctxDone := make(chan os.Signal, 1)
		signal.Notify(ctxDone, syscall.SIGINT, syscall.SIGTERM)
		<-ctxDone
		logger.Info("shutting down worker")
		w.Shutdown()
	}()

	if err := w.Start(); err != nil {
		logger.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
