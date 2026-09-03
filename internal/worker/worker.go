package worker

import (
	"log/slog"

	"github.com/hibiken/asynq"
)

type Worker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

func New(redisAddr string) *Worker {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	mux := asynq.NewServeMux()
	// Task handlers (ingest, embed, index) are registered here.

	return &Worker{server: srv, mux: mux}
}

func (w *Worker) Start() error {
	slog.Info("asynq worker starting")
	return w.server.Run(w.mux)
}

func (w *Worker) Shutdown() {
	w.server.Shutdown()
}
