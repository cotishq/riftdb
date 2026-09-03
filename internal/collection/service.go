package collection

import (
	"context"
	"errors"
	"strings"
)

var ErrInvalidName = errors.New("collection name is required")

type Service interface {
	Create(ctx context.Context, name string, description *string) (*Collection, error)
	Get(ctx context.Context, id string) (*Collection, error)
	List(ctx context.Context) ([]Collection, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, name string, description *string) (*Collection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidName
	}
	if description != nil {
		trimmed := strings.TrimSpace(*description)
		if trimmed == "" {
			description = nil
		} else {
			description = &trimmed
		}
	}
	return s.repo.Create(ctx, name, description)
}

func (s *service) Get(ctx context.Context, id string) (*Collection, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context) ([]Collection, error) {
	return s.repo.List(ctx)
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
