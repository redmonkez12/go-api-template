.PHONY: help run build test docker-up docker-down migrate-up migrate-down migrate-create swagger docker-build docker-run docker-prod-run

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

run: ## Run the application
	go run cmd/api/main.go

build: ## Build the application
	go build -o bin/api cmd/api/main.go

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

docker-up: ## Start Docker containers
	docker-compose up -d

docker-down: ## Stop Docker containers
	docker-compose down

docker-logs: ## View Docker container logs
	docker-compose logs -f

migrate-up: ## Run database migrations up
	@echo "Running migrations..."
	@set -a && . ./.env && set +a && migrate -path migrations -database "postgres://$${DB_USER}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT}/$${DB_NAME}?sslmode=$${DB_SSLMODE}" up

migrate-down: ## Run database migrations down
	@echo "Rolling back migrations..."
	@set -a && . ./.env && set +a && migrate -path migrations -database "postgres://$${DB_USER}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT}/$${DB_NAME}?sslmode=$${DB_SSLMODE}" down

migrate-create: ## Create a new migration (usage: make migrate-create NAME=migration_name)
	@migrate create -ext sql -dir migrations -seq $(NAME)

deps: ## Download dependencies
	go mod download
	go mod tidy

install-tools: ## Install development tools
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/swaggo/swag/cmd/swag@latest

swagger: ## Generate Swagger documentation
	@echo "Generating Swagger documentation..."
	@$(shell go env GOPATH)/bin/swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
	@echo "Swagger documentation generated in docs/"

swagger-clean: ## Clean generated Swagger files
	@echo "Cleaning Swagger documentation..."
	@rm -rf docs/

docker-build: ## Build production Docker image
	@echo "Building production Docker image..."
	@docker build -t go-api-template:latest .

docker-run: ## Run production Docker image (requires .env file or env vars)
	@echo "Running production Docker container..."
	@docker run --rm \
		--name go-api-template \
		-p 8080:8080 \
		--env-file .env \
		-e APP_ENV=prod \
		go-api-template:latest

docker-dev-run: ## Run Docker image in development mode
	@echo "Running development Docker container with Swagger enabled..."
	@docker run --rm \
		--name go-api-template-dev \
		-p 8080:8080 \
		--env-file .env \
		-e APP_ENV=dev \
		go-api-template:latest

docker-stop: ## Stop running Docker container
	@docker stop go-api-template || true
	@docker stop go-api-template-dev || true
