SHELL := /bin/bash
GO    ?= go
APP   := realtime-chat
PKG   := ./...

DB_URL ?= postgres://chat:chatpw@localhost:5432/chat?sslmode=disable

.PHONY: help build run tidy fmt vet test docker-up docker-down migrate-up migrate-down

help:
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

build: ## build the server binary
	$(GO) build -o bin/server ./cmd/server

run: ## run the server locally
	$(GO) run ./cmd/server

tidy: ## go mod tidy
	$(GO) mod tidy

fmt: ## gofmt
	$(GO) fmt $(PKG)

vet: ## go vet
	$(GO) vet $(PKG)

test: ## go test
	$(GO) test $(PKG)

docker-up: ## start postgres, redis, server via docker compose
	docker compose -f deployments/docker-compose.yml up --build

docker-down: ## stop docker compose stack
	docker compose -f deployments/docker-compose.yml down -v

migrate-up: ## apply migrations against $(DB_URL)
	migrate -path migrations -database "$(DB_URL)" up

migrate-down: ## roll migrations back
	migrate -path migrations -database "$(DB_URL)" down
