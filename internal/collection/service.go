package collection

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

const defaultEmbeddingModel = "nomic-embed-text"

var (
	ErrInvalidName       = errors.New("collection name is required")
	ErrInvalidNameFormat = errors.New("collection name must be a DNS label: lowercase letters, digits, hyphens")
)

var namePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

type Service interface {
	Create(ctx context.Context, in CreateInput) (*Collection, error)
	Get(ctx context.Context, id string) (*Collection, error)
	List(ctx context.Context) ([]Collection, error)
	Delete(ctx context.Context, id string) error
}

type CreateInput struct {
	Name           string
	Description    *string
	EmbeddingModel string
	Namespace      string
	SourcePrefix   string
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, in CreateInput) (*Collection, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, ErrInvalidName
	}
	if !namePattern.MatchString(in.Name) {
		return nil, ErrInvalidNameFormat
	}

	if in.Description != nil {
		trimmed := strings.TrimSpace(*in.Description)
		if trimmed == "" {
			in.Description = nil
		} else {
			in.Description = &trimmed
		}
	}

	in.EmbeddingModel = strings.TrimSpace(in.EmbeddingModel)
	if in.EmbeddingModel == "" {
		in.EmbeddingModel = defaultEmbeddingModel
	}

	in.Namespace = strings.TrimSpace(in.Namespace)
	if in.Namespace == "" {
		in.Namespace = in.Name
	}

	in.SourcePrefix = strings.Trim(strings.TrimSpace(in.SourcePrefix), "/")
	if in.SourcePrefix == "" {
		in.SourcePrefix = "collections/" + in.Name
	}

	return s.repo.Create(ctx, CreateParams{
		Name:           in.Name,
		Description:    in.Description,
		EmbeddingModel: in.EmbeddingModel,
		Namespace:      in.Namespace,
		SourcePrefix:   in.SourcePrefix,
	})
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
