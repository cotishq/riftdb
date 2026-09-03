package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cotishq/riftdb/internal/collection"
	"github.com/cotishq/riftdb/internal/document"
)

func NewRouter(pool *pgxpool.Pool, collections *collection.Handler, documents *document.Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", healthHandler(pool))

	r.Route("/v1", func(r chi.Router) {
		r.Route("/collections", func(r chi.Router) {
			r.Post("/", collections.Create)
			r.Get("/", collections.List)
			r.Get("/{id}", collections.Get)
			r.Delete("/{id}", collections.Delete)

			r.Post("/{id}/documents", documents.Create)
			r.Get("/{id}/documents", documents.List)
		})

		r.Route("/documents", func(r chi.Router) {
			r.Get("/{id}", documents.Get)
			r.Delete("/{id}", documents.Delete)
		})
	})

	return r
}

func healthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		status := http.StatusOK
		body := map[string]string{"status": "ok"}

		if err := pool.Ping(ctx); err != nil {
			status = http.StatusServiceUnavailable
			body = map[string]string{"status": "degraded", "error": "database unreachable"}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}
}
