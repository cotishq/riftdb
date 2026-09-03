package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidCollection = errors.New("collection id is required")
	ErrInvalidR2Key      = errors.New("r2_key is required")
)

type Service interface {
	Create(ctx context.Context, collectionID, contentHash, r2Key string) (*Document, error)
	Get(ctx context.Context, id string) (*Document, error)
	List(ctx context.Context, collectionID string) ([]Document, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, collectionID, contentHash, r2Key string) (*Document, error) {
	collectionID = strings.TrimSpace(collectionID)
	r2Key = strings.TrimSpace(r2Key)
	contentHash = strings.TrimSpace(contentHash)

	if collectionID == "" {
		return nil, ErrInvalidCollection
	}
	if r2Key == "" {
		return nil, ErrInvalidR2Key
	}
	if contentHash == "" {
		sum := sha256.Sum256([]byte(r2Key))
		contentHash = hex.EncodeToString(sum[:])
	}

	d, err := s.repo.Create(ctx, CreateParams{
		CollectionID: collectionID,
		ContentHash:  contentHash,
		R2Key:        r2Key,
	})
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}
	return d, nil
}

func (s *service) Get(ctx context.Context, id string) (*Document, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context, collectionID string) ([]Document, error) {
	if strings.TrimSpace(collectionID) == "" {
		return nil, ErrInvalidCollection
	}
	return s.repo.ListByCollection(ctx, collectionID)
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
