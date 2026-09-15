package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/cotishq/riftdb/internal/document"
)

const TaskIngest = "document:ingest"

type Payload struct {
	DocumentID   string `json:"document_id"`
	CollectionID string `json:"collection_id"`
	R2Key        string `json:"r2_key"`
}

type Client struct {
	asynq *asynq.Client
}

func NewClient(redisAddr string) *Client {
	return &Client{asynq: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})}
}

func (c *Client) Close() error {
	return c.asynq.Close()
}

func (c *Client) EnqueuePending(ctx context.Context, doc document.Document) error {
	if doc.ID == "" {
		return errors.New("document id is required")
	}
	body, err := json.Marshal(Payload{
		DocumentID:   doc.ID,
		CollectionID: doc.CollectionID,
		R2Key:        doc.R2Key,
	})
	if err != nil {
		return fmt.Errorf("encode ingest payload: %w", err)
	}

	task := asynq.NewTask(TaskIngest, body)
	_, err = c.asynq.EnqueueContext(ctx, task,
		asynq.TaskID(doc.ID),
		asynq.Queue("default"),
	)
	if err != nil {
		if errors.Is(err, asynq.ErrTaskIDConflict) || errors.Is(err, asynq.ErrDuplicateTask) {
			return nil
		}
		return fmt.Errorf("enqueue ingest %s: %w", doc.ID, err)
	}
	return nil
}
