.PHONY: run build migrate tidy

APP := riftdb
BIN := bin/api
DATABASE_URL ?= postgres://riftdb:riftdb@localhost:5432/riftdb?sslmode=disable

run:
	go run ./cmd/api

build:
	mkdir -p bin
	go build -o $(BIN) ./cmd/api

migrate:
	docker compose exec -T postgres psql -U riftdb -d riftdb -f /docker-entrypoint-initdb.d/001_init.sql

tidy:
	go mod tidy
