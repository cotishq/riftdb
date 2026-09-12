package reconcile

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/cotishq/riftdb/internal/collection"
	"github.com/cotishq/riftdb/internal/document"
	"github.com/cotishq/riftdb/internal/storage"
)

type fakeDocs struct {
	mu      sync.Mutex
	byColl  map[string][]document.Document
	creates int
}

func newFakeDocs() *fakeDocs {
	return &fakeDocs{byColl: make(map[string][]document.Document)}
}

func (f *fakeDocs) List(_ context.Context, collectionID string) ([]document.Document, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]document.Document, len(f.byColl[collectionID]))
	copy(out, f.byColl[collectionID])
	return out, nil
}

func (f *fakeDocs) Create(_ context.Context, collectionID, contentHash, r2Key string) (*document.Document, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range f.byColl[collectionID] {
		if d.R2Key == r2Key {
			return nil, document.ErrConflict
		}
	}
	f.creates++
	d := document.Document{
		ID:           "doc-" + r2Key,
		CollectionID: collectionID,
		ContentHash:  contentHash,
		R2Key:        r2Key,
		Status:       "pending",
	}
	f.byColl[collectionID] = append(f.byColl[collectionID], d)
	return &d, nil
}

func papers() collection.Collection {
	return collection.Collection{
		ID:           "papers-uuid",
		Name:         "papers",
		SourcePrefix: "collections/papers",
	}
}

func TestReconcileInsertsPendingForNewObject(t *testing.T) {
	ctx := context.Background()
	mem := storage.NewMemory()
	if err := mem.Put(ctx, "collections/papers/notes.txt", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	docs := newFakeDocs()
	r := New(mem, docs)

	n, err := r.Reconcile(ctx, papers())
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if n != 1 {
		t.Fatalf("inserted: got %d, want 1", n)
	}

	listed, err := docs.List(ctx, "papers-uuid")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("documents: got %d, want 1", len(listed))
	}
	if listed[0].R2Key != "collections/papers/notes.txt" {
		t.Fatalf("r2_key: %q", listed[0].R2Key)
	}
	if listed[0].Status != "pending" {
		t.Fatalf("status: %q", listed[0].Status)
	}
}

func TestReconcileIsIdempotent(t *testing.T) {
	ctx := context.Background()
	mem := storage.NewMemory()
	if err := mem.Put(ctx, "collections/papers/notes.txt", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	docs := newFakeDocs()
	r := New(mem, docs)

	if _, err := r.Reconcile(ctx, papers()); err != nil {
		t.Fatalf("first: %v", err)
	}
	n, err := r.Reconcile(ctx, papers())
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if n != 0 {
		t.Fatalf("second inserted: got %d, want 0", n)
	}
	if docs.creates != 1 {
		t.Fatalf("creates: got %d, want 1", docs.creates)
	}

	listed, err := docs.List(ctx, "papers-uuid")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("documents: got %d, want 1", len(listed))
	}
}

func TestReconcileIgnoresOtherPrefix(t *testing.T) {
	ctx := context.Background()
	mem := storage.NewMemory()
	for _, key := range []string{
		"collections/other/notes.txt",
		"collections/papers-old/notes.txt",
		"collections/papers/",
	} {
		if err := mem.Put(ctx, key, []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	docs := newFakeDocs()
	r := New(mem, docs)

	n, err := r.Reconcile(ctx, papers())
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if n != 0 {
		t.Fatalf("inserted: got %d, want 0", n)
	}
	listed, err := docs.List(ctx, "papers-uuid")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 0 {
		t.Fatalf("documents: got %d, want 0", len(listed))
	}
}

func TestReconcileRequiresCollectionAndPrefix(t *testing.T) {
	r := New(storage.NewMemory(), newFakeDocs())
	ctx := context.Background()

	if _, err := r.Reconcile(ctx, collection.Collection{SourcePrefix: "collections/papers"}); !errors.Is(err, ErrInvalidCollection) {
		t.Fatalf("missing id: got %v", err)
	}
	if _, err := r.Reconcile(ctx, collection.Collection{ID: "papers-uuid"}); !errors.Is(err, ErrInvalidPrefix) {
		t.Fatalf("missing prefix: got %v", err)
	}
}

func others() collection.Collection {
	return collection.Collection{
		ID:           "other-uuid",
		Name:         "other",
		SourcePrefix: "collections/other",
	}
}

func TestReconcileAllInsertsPerCollection(t *testing.T) {
	ctx := context.Background()
	mem := storage.NewMemory()
	if err := mem.Put(ctx, "collections/papers/notes.txt", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := mem.Put(ctx, "collections/other/notes.txt", []byte("other")); err != nil {
		t.Fatal(err)
	}
	docs := newFakeDocs()
	r := New(mem, docs)

	n, err := r.ReconcileAll(ctx, []collection.Collection{papers(), others()})
	if err != nil {
		t.Fatalf("ReconcileAll: %v", err)
	}
	if n != 2 {
		t.Fatalf("inserted: got %d, want 2", n)
	}
	for _, id := range []string{"papers-uuid", "other-uuid"} {
		listed, err := docs.List(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if len(listed) != 1 {
			t.Fatalf("%s documents: got %d, want 1", id, len(listed))
		}
	}
}

func TestReconcileAllEmpty(t *testing.T) {
	n, err := New(storage.NewMemory(), newFakeDocs()).ReconcileAll(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("inserted: got %d, want 0", n)
	}
}

func TestRunOnceListsCollections(t *testing.T) {
	ctx := context.Background()
	mem := storage.NewMemory()
	if err := mem.Put(ctx, "collections/papers/notes.txt", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	docs := newFakeDocs()
	r := New(mem, docs)

	n, err := r.RunOnce(ctx, func(context.Context) ([]collection.Collection, error) {
		return []collection.Collection{papers()}, nil
	})
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if n != 1 {
		t.Fatalf("inserted: got %d, want 1", n)
	}
}

func TestRunOnceRequiresLister(t *testing.T) {
	_, err := New(storage.NewMemory(), newFakeDocs()).RunOnce(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
