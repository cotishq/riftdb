package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubService struct {
	in  CreateInput
	out *Collection
	err error
}

func (s *stubService) Create(_ context.Context, in CreateInput) (*Collection, error) {
	s.in = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &Collection{
		ID:             "11111111-1111-1111-1111-111111111111",
		Name:           in.Name,
		EmbeddingModel: "nomic-embed-text",
		Namespace:      in.Name,
		SourcePrefix:   "collections/" + in.Name,
		CreatedAt:      time.Unix(0, 0).UTC(),
		UpdatedAt:      time.Unix(0, 0).UTC(),
	}, nil
}

func (s *stubService) Get(context.Context, string) (*Collection, error) { return nil, ErrNotFound }
func (s *stubService) List(context.Context) ([]Collection, error)       { return nil, nil }
func (s *stubService) Delete(context.Context, string) error             { return nil }

func TestCreateHandlerCreated(t *testing.T) {
	h := NewHandler(&stubService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/collections", bytes.NewBufferString(`{"name":"papers","description":"research"}`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d body %s", rec.Code, rec.Body.String())
	}
	var got Collection
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "papers" || got.ID == "" {
		t.Fatalf("unexpected collection: %+v", got)
	}
}

func TestCreateHandlerBadJSON(t *testing.T) {
	h := NewHandler(&stubService{})
	req := httptest.NewRequest(http.MethodPost, "/v1/collections", bytes.NewBufferString(`{`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestCreateHandlerConflict(t *testing.T) {
	h := NewHandler(&stubService{err: ErrConflict})
	req := httptest.NewRequest(http.MethodPost, "/v1/collections", bytes.NewBufferString(`{"name":"papers"}`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d body %s", rec.Code, rec.Body.String())
	}
}
