package collection

import (
	"context"
	"testing"
	"time"
)

type stubRepo struct {
	last CreateParams
	err  error
}

func (s *stubRepo) Create(_ context.Context, p CreateParams) (*Collection, error) {
	s.last = p
	if s.err != nil {
		return nil, s.err
	}
	return &Collection{
		ID:             "00000000-0000-0000-0000-000000000001",
		Name:           p.Name,
		Description:    p.Description,
		EmbeddingModel: p.EmbeddingModel,
		Namespace:      p.Namespace,
		SourcePrefix:   p.SourcePrefix,
		CreatedAt:      time.Unix(0, 0).UTC(),
		UpdatedAt:      time.Unix(0, 0).UTC(),
	}, nil
}

func (s *stubRepo) GetByID(context.Context, string) (*Collection, error) { return nil, ErrNotFound }
func (s *stubRepo) List(context.Context) ([]Collection, error)           { return nil, nil }
func (s *stubRepo) Delete(context.Context, string) error                 { return nil }

func TestCreateAppliesDefaults(t *testing.T) {
	repo := &stubRepo{}
	svc := NewService(repo)

	got, err := svc.Create(context.Background(), CreateInput{Name: " papers "})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.EmbeddingModel != defaultEmbeddingModel {
		t.Fatalf("embedding_model: got %q", got.EmbeddingModel)
	}
	if got.Namespace != "papers" {
		t.Fatalf("namespace: got %q", got.Namespace)
	}
	if got.SourcePrefix != "collections/papers" {
		t.Fatalf("source_prefix: got %q", got.SourcePrefix)
	}
}

func TestCreateRejectsInvalidName(t *testing.T) {
	svc := NewService(&stubRepo{})

	if _, err := svc.Create(context.Background(), CreateInput{Name: ""}); err != ErrInvalidName {
		t.Fatalf("empty name: got %v", err)
	}
	if _, err := svc.Create(context.Background(), CreateInput{Name: "Papers"}); err != ErrInvalidNameFormat {
		t.Fatalf("uppercase name: got %v", err)
	}
}

func TestCreateConflict(t *testing.T) {
	svc := NewService(&stubRepo{err: ErrConflict})
	if _, err := svc.Create(context.Background(), CreateInput{Name: "papers"}); err != ErrConflict {
		t.Fatalf("conflict: got %v", err)
	}
}
