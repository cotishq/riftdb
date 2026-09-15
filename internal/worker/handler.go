package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

func HandleIngest(_ context.Context, t *asynq.Task) error {
	var p Payload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("decode ingest payload: %w", err)
	}
	if p.DocumentID == "" {
		return fmt.Errorf("ingest payload missing document_id")
	}
	slog.Info("ingest job",
		"document_id", p.DocumentID,
		"collection_id", p.CollectionID,
		"r2_key", p.R2Key,
	)
	return nil
}
