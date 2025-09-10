.PHONY: build run test clean generate docker docker-up docker-down

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build parameters
BINARY_NAME=vaultwarden-syncer
BINARY_UNIX=$(BINARY_NAME)_unix

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/server

build-linux:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) ./cmd/server

run:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/server
	./$(BINARY_NAME)

test:
	$(GOTEST) -v ./...

test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f coverage.out
	rm -f coverage.html

generate:
	$(GOCMD) run -mod=mod entgo.io/ent/cmd/ent generate ./ent/schema

deps:
	$(GOMOD) download
	$(GOMOD) verify

tidy:
	$(GOMOD) tidy

docker:
	docker build -t vaultwarden-syncer .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f vaultwarden-syncer

dev:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/server
	./$(BINARY_NAME) &
	PID=$$!; \
	echo "Server started with PID $$PID"; \
	trap "kill $$PID" EXIT; \
	while inotifywait -e modify -r .; do \
		kill $$PID; \
		$(GOBUILD) -o $(BINARY_NAME) ./cmd/server; \
		./$(BINARY_NAME) & \
		PID=$$!; \
	done