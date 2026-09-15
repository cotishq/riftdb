.PHONY: run build migrate tidy list-prefix reconcile worker

APP := riftdb
BIN := bin/api
DATABASE_URL ?= postgres://riftdb:riftdb@localhost:5432/riftdb?sslmode=disable
PREFIX ?= collections/papers

run:
	go run ./cmd/api

build:
	mkdir -p bin
	go build -o $(BIN) ./cmd/api
	go build -o bin/s3list ./cmd/s3list
	go build -o bin/reconcile ./cmd/reconcile
	go build -o bin/worker ./cmd/worker

list-prefix:
	go run ./cmd/s3list $(PREFIX)

reconcile:
	go run ./cmd/reconcile

worker:
	go run ./cmd/worker

migrate:
	docker compose exec -T postgres psql -U riftdb -d riftdb -f /docker-entrypoint-initdb.d/001_init.sql
	docker compose exec -T postgres psql -U riftdb -d riftdb -f /docker-entrypoint-initdb.d/002_collection_sync.sql
	docker compose exec -T postgres psql -U riftdb -d riftdb -f /docker-entrypoint-initdb.d/003_documents_r2_key_unique.sql

tidy:
	go mod tidy
