package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hibiken/asynq"
)

func TestHandleIngest(t *testing.T) {
	body, err := json.Marshal(Payload{
		DocumentID:   "11111111-1111-1111-1111-111111111111",
		CollectionID: "papers-uuid",
		R2Key:        "collections/papers/notes.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	task := asynq.NewTask(TaskIngest, body)
	if err := HandleIngest(context.Background(), task); err != nil {
		t.Fatalf("HandleIngest: %v", err)
	}
}

func TestHandleIngestRejectsBadPayload(t *testing.T) {
	task := asynq.NewTask(TaskIngest, []byte(`{}`))
	if err := HandleIngest(context.Background(), task); err == nil {
		t.Fatal("expected error")
	}
}
