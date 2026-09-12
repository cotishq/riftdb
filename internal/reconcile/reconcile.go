package reconcile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cotishq/riftdb/internal/collection"
	"github.com/cotishq/riftdb/internal/document"
	"github.com/cotishq/riftdb/internal/storage"
)

type CollectionLister func(context.Context) ([]collection.Collection, error)

var (
	ErrInvalidCollection = errors.New("collection id is required")
	ErrInvalidPrefix     = errors.New("source prefix is required")
)

type documents interface {
	Create(ctx context.Context, collectionID, contentHash, r2Key string) (*document.Document, error)
	List(ctx context.Context, collectionID string) ([]document.Document, error)
}

type objects interface {
	List(ctx context.Context, prefix string) ([]storage.Object, error)
}

// Reconciler records objects under a collection prefix that have no ledger row.
type Reconciler struct {
	objects   objects
	documents documents
}

func New(objects objects, documents documents) *Reconciler {
	return &Reconciler{objects: objects, documents: documents}
}

// Reconcile inserts a pending documents row for each new object under coll.SourcePrefix.
// It does not download, embed, or enqueue work. Returns how many rows were inserted.
func (r *Reconciler) Reconcile(ctx context.Context, coll collection.Collection) (int, error) {
	if strings.TrimSpace(coll.ID) == "" {
		return 0, ErrInvalidCollection
	}
	prefix := strings.Trim(strings.TrimSpace(coll.SourcePrefix), "/")
	if prefix == "" {
		return 0, ErrInvalidPrefix
	}

	listed, err := r.objects.List(ctx, prefix)
	if err != nil {
		return 0, fmt.Errorf("list objects: %w", err)
	}

	existing, err := r.documents.List(ctx, coll.ID)
	if err != nil {
		return 0, fmt.Errorf("list documents: %w", err)
	}

	seen := make(map[string]struct{}, len(existing))
	for _, d := range existing {
		seen[d.R2Key] = struct{}{}
	}

	inserted := 0
	for _, obj := range listed {
		if !underPrefix(obj.Key, prefix) {
			continue
		}
		if _, ok := seen[obj.Key]; ok {
			continue
		}

		_, err := r.documents.Create(ctx, coll.ID, "", obj.Key)
		if err != nil {
			if errors.Is(err, document.ErrConflict) {
				seen[obj.Key] = struct{}{}
				continue
			}
			return inserted, fmt.Errorf("create document %s: %w", obj.Key, err)
		}
		seen[obj.Key] = struct{}{}
		inserted++
	}

	return inserted, nil
}

// ReconcileAll runs Reconcile for each collection. Stops on the first error.
func (r *Reconciler) ReconcileAll(ctx context.Context, collections []collection.Collection) (int, error) {
	inserted := 0
	for _, coll := range collections {
		n, err := r.Reconcile(ctx, coll)
		inserted += n
		if err != nil {
			return inserted, fmt.Errorf("collection %s: %w", coll.ID, err)
		}
	}
	return inserted, nil
}

// RunOnce lists collections and records any missing objects.
func (r *Reconciler) RunOnce(ctx context.Context, list CollectionLister) (int, error) {
	if list == nil {
		return 0, errors.New("collection lister is required")
	}
	collections, err := list(ctx)
	if err != nil {
		return 0, fmt.Errorf("list collections: %w", err)
	}
	return r.ReconcileAll(ctx, collections)
}

// Loop runs RunOnce immediately, then on every interval until ctx is cancelled.
func (r *Reconciler) Loop(ctx context.Context, interval time.Duration, list CollectionLister) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	run := func() {
		n, err := r.RunOnce(ctx, list)
		if err != nil {
			slog.Error("reconcile failed", "error", err, "inserted", n)
			return
		}
		slog.Info("reconcile complete", "inserted", n)
	}

	run()
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

func underPrefix(key, prefix string) bool {
	key = strings.TrimSpace(key)
	if key == "" || strings.HasSuffix(key, "/") {
		return false
	}
	return key == prefix || strings.HasPrefix(key, prefix+"/")
}
