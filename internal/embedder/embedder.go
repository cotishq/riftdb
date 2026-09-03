package embedder

import "context"

// Embedder turns text into dense vectors for a specific model.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	ModelID() string
}
