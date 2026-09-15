# RiftDB

Object storage is the source of truth. RiftDB watches a collection prefix and records what it sees. The vector index is the eventual destination. It is not a search engine yet.

## Now

```mermaid
flowchart LR
  You["You drop a file"] --> MinIO[(MinIO / R2)]
  API["POST /v1/collections"] --> PG[(Postgres collections)]
  Tick["Ticker / make reconcile"] --> PG
  Tick --> MinIO
  Tick -->|"insert pending if key is new"| Docs[(Postgres documents)]
  Tick -->|enqueue if pending| Redis[(Redis / Asynq)]
  Redis --> W["worker logs document id"]
```

A collection is a watch spec (`source_prefix`, embedding model, namespace). Every `RECONCILE_INTERVAL` (default 30s) the API lists that prefix and inserts a `documents` row when the key is missing. A unique index on `(collection_id, r2_key)` stops duplicates. Pending rows are enqueued to Asynq (task id = document UUID). The worker currently only logs the job.

`POST /v1/collections/{id}/documents` still exists. Discovery does not use it.

## Later

```mermaid
flowchart LR
  MinIO[(object storage)] --> Rec[reconcile]
  Rec --> PG[(ledger)]
  PG -->|pending| Worker[worker]
  Worker --> Chunk[chunk + embed]
  Chunk --> TPUF[(Turbopuffer)]
  Q["GET query"] --> TPUF
```

Download, chunk, embed, upsert, and query are not wired. The worker does not read the object yet.

## Run

```bash
cp .env.example .env
docker compose up -d postgres minio minio-init redis
make run
```

API: `http://localhost:8080`  
MinIO console: `http://localhost:9001` (user `riftdb` / `riftdbsecret`)

```bash
curl -sS -X POST localhost:8080/v1/collections \
  -H 'Content-Type: application/json' \
  -d '{"name":"papers"}'

# put an object at collections/papers/notes.txt in bucket riftdb
make reconcile
curl -sS localhost:8080/v1/collections
```

`make list-prefix` lists MinIO. `make migrate` applies SQL if the volume is old.

Redis is in compose for Asynq. `make run` starts the worker in-process. `make worker` runs it standalone. Port 6379 / 8080 conflicts are host problems, not app bugs.

## Layout

| Path | Role |
| --- | --- |
| `cmd/api` | HTTP + reconcile loop + worker |
| `cmd/reconcile` | one-shot discovery |
| `cmd/s3list` | list a prefix |
| `cmd/worker` | standalone Asynq worker |
| `internal/reconcile` | diff prefix vs ledger, enqueue pending |
| `internal/worker` | ingest job (log only) |
| `internal/storage` | S3 + in-memory |
| `migrations/` | Postgres |
