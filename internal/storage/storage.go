package storage

import "context"

// Storage is object storage for raw document payloads (R2/S3-compatible).
type Storage interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}
