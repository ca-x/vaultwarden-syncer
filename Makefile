.PHONY: help build clean test run dev docker-build docker-run setup generate fmt lint

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME=vaultwarden-syncer
BUILD_DIR=./build
DOCKER_IMAGE=vaultwarden-syncer
DOCKER_TAG=latest

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Install dependencies and tools
	go mod download
	go install entgo.io/ent/cmd/ent@latest

generate: ## Generate Ent code
	ent generate ./ent/schema

build: generate ## Build the application
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags '-w -s' -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	go clean

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

run: build ## Run the application
	$(BUILD_DIR)/$(BINARY_NAME)

dev: generate ## Run in development mode (with auto-reload)
	go run ./cmd/server

fmt: ## Format code
	go fmt ./...
	goimports -w .

lint: ## Run linter
	golangci-lint run

# Database operations
db-reset: ## Reset database (delete and recreate)
	rm -f ./data/syncer.db
	mkdir -p ./data

# Docker operations
docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## Run Docker container
	docker run -p 8181:8181 -v $(PWD)/config.yaml:/app/config.yaml:ro $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-compose-up: ## Start with docker-compose
	docker-compose up -d

docker-compose-down: ## Stop docker-compose services
	docker-compose down

docker-compose-logs: ## View docker-compose logs
	docker-compose logs -f

# Development helpers
init-config: ## Create initial config file from example
	cp config.yaml.example config.yaml
	@echo "Config file created. Please edit config.yaml with your settings."

create-dirs: ## Create necessary directories
	mkdir -p data logs data/vaultwarden

install: build ## Install the binary to system
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

uninstall: ## Remove the binary from system
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

# Release operations
release: clean test build ## Build release version
	@echo "Release build completed in $(BUILD_DIR)/"

all: clean setup generate test build ## Run all main tasks

# Development workflow
dev-setup: setup generate init-config create-dirs ## Complete development setup
	@echo "Development environment is ready!"
	@echo "1. Edit config.yaml with your settings"
	@echo "2. Run 'make dev' to start development server"