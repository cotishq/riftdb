package document

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("document not found")

type Document struct {
	ID           string    `json:"id"`
	CollectionID string    `json:"collection_id"`
	ContentHash  string    `json:"content_hash"`
	R2Key        string    `json:"r2_key"`
	Status       string    `json:"status"`
	ModelID      *string   `json:"model_id,omitempty"`
	ChunkCount   int       `json:"chunk_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateParams struct {
	CollectionID string
	ContentHash  string
	R2Key        string
}

type Repository interface {
	Create(ctx context.Context, p CreateParams) (*Document, error)
	GetByID(ctx context.Context, id string) (*Document, error)
	ListByCollection(ctx context.Context, collectionID string) ([]Document, error)
	Delete(ctx context.Context, id string) error
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Create(ctx context.Context, p CreateParams) (*Document, error) {
	const q = `
		INSERT INTO documents (collection_id, content_hash, r2_key)
		VALUES ($1, $2, $3)
		RETURNING id, collection_id, content_hash, r2_key, status, model_id, chunk_count, created_at, updated_at`

	d, err := scanDocument(r.pool.QueryRow(ctx, q, p.CollectionID, p.ContentHash, p.R2Key))
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}
	return d, nil
}

func (r *repository) GetByID(ctx context.Context, id string) (*Document, error) {
	const q = `
		SELECT id, collection_id, content_hash, r2_key, status, model_id, chunk_count, created_at, updated_at
		FROM documents
		WHERE id = $1`

	d, err := scanDocument(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return d, nil
}

func (r *repository) ListByCollection(ctx context.Context, collectionID string) ([]Document, error) {
	const q = `
		SELECT id, collection_id, content_hash, r2_key, status, model_id, chunk_count, created_at, updated_at
		FROM documents
		WHERE collection_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q, collectionID)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	out := make([]Document, 0)
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		out = append(out, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	return out, nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanDocument(s scanner) (*Document, error) {
	var d Document
	if err := s.Scan(
		&d.ID,
		&d.CollectionID,
		&d.ContentHash,
		&d.R2Key,
		&d.Status,
		&d.ModelID,
		&d.ChunkCount,
		&d.CreatedAt,
		&d.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &d, nil
}
