.PHONY: build test lint vet coverage clean

VERSION ?= dev
LDFLAGS := -X github.com/mattiasthalen/qlik-sync/cmd.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o qs .

test:
	go test -race ./...

lint:
	golangci-lint run

vet:
	go vet ./...

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -f qs coverage.out
