SHELL := /bin/bash

.PHONY: build run test fmt vet lint docker-build docker-run

build: fmt
	go build -o bin/pet ./cmd/server

run:
	REDIS_PASSWORD=$(REDIS_PASSWORD) POSTGRES_DSN=$(POSTGRES_DSN) go run ./cmd/server

test:
	go test ./... -v

fmt:
	gofmt -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

docker-build:
	docker build -t go-task-manager:latest .

docker-run:
	docker run --rm -p 8080:8080 -e REDIS_PASSWORD=$(REDIS_PASSWORD) -e POSTGRES_DSN=$(POSTGRES_DSN) go-task-manager:latest
