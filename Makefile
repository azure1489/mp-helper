.PHONY: build server cli test lint tidy

build: server cli

server:
	go build -o bin/mp-server ./cmd/mp-server

cli:
	go build -o bin/mp-cli ./cmd/mp-cli

test:
	go test ./...

tidy:
	go mod tidy

lint:
	go vet ./...
