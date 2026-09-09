package storage

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("object not found")

// Object is a listed object in storage. Payload is not included.
type Object struct {
	Key string
}

// Storage is object storage for raw document payloads (R2/S3-compatible).
type Storage interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]Object, error)
}
