.PHONY: build run fmt vet test race tidy compose-up compose-down

build:
	go build ./...

run:
	go run ./cmd/api

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

tidy:
	go mod tidy

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down -v
