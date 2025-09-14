.PHONY: build run test clean docker-build docker-run docker-stop fmt lint deps help

# Variables
APP_NAME=joinly-manager
DOCKER_IMAGE=$(APP_NAME):latest
GO_VERSION=1.24

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development
build: ## Build the application
	go build -o bin/$(APP_NAME) ./cmd/server

run: ## Run the application
	go run ./cmd/server/main.go

test: ## Run tests
	go test ./...

clean: ## Clean build artifacts
	rm -rf bin/
	go clean

fmt: ## Format code
	go fmt ./...

lint: ## Run linter
	golangci-lint run

deps: ## Download dependencies
	go mod download
	go mod tidy

# Docker
docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run Docker container
	docker run -p 8001:8001 --name $(APP_NAME) $(DOCKER_IMAGE)

docker-stop: ## Stop Docker container
	docker stop $(APP_NAME)
	docker rm $(APP_NAME)

docker-compose-up: ## Start with docker-compose
	docker-compose up --build

docker-compose-down: ## Stop docker-compose
	docker-compose down

# Development setup
dev-setup: ## Setup development environment
	go mod download
	@echo "Development environment setup complete"
	@echo "Run 'make run' to start the server"

# Production build
prod-build: ## Build for production
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/$(APP_NAME) ./cmd/server

# Cross-platform builds
build-linux: ## Build for Linux
	GOOS=linux GOARCH=amd64 go build -o bin/$(APP_NAME)-linux ./cmd/server

build-windows: ## Build for Windows
	GOOS=windows GOARCH=amd64 go build -o bin/$(APP_NAME)-windows.exe ./cmd/server

build-darwin: ## Build for macOS
	GOOS=darwin GOARCH=amd64 go build -o bin/$(APP_NAME)-darwin ./cmd/server

build-all: build-linux build-windows build-darwin ## Build for all platforms

# Utilities
logs: ## Show application logs (when running in Docker)
	docker-compose logs -f $(APP_NAME)

health-check: ## Check application health
	curl -f http://localhost:8001/ || echo "Health check failed"

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# CI/CD
ci-test: deps ## Run CI tests
	go test -race -coverprofile=coverage.out ./...

ci-build: ## CI build
	go build -v ./cmd/server

# Database (if needed in future)
# migrate-up: ## Run database migrations
# migrate-down: ## Rollback database migrations

# Deployment
deploy: docker-build ## Deploy to production
	@echo "Deploy $(DOCKER_IMAGE) to your container registry/orchestrator"
