## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	@echo '-- Running application --'
	@docker compose exec go run ./cmd/api -port=8081 -db-dsn=postgres://user:password@postgres/mydb?sslmode=disable -cors-trusted-origins="http://localhost:8080"

## db/seed: generate password and update seed file
.PHONY: db/seed
db/seed:
	@echo "Generating password hash and updating seed file..."
	@docker compose exec go sh -c 'go run -c '\''package main; import "golang.org/x/crypto/bcrypt"; import "fmt"; func main() { hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost); fmt.Println(string(hash)) }'\' > /tmp/hash.txt
	@docker compose exec go sh -c 'HASH=$$(cat /tmp/hash.txt) && sed -i "s|<bcrypt_hash_here>|$$HASH|g" /migrations/002_seed_roles_users.up.sql'
	@echo "Seed file updated with generated hash"

.PHONY: migrate/up
migrate/up:
	@docker run --rm -v ./app/migrations:/migrations --network podman-docker-compose-files_backend migrate/migrate \
		-path=/migrations -database "postgres://user:password@postgres/mydb?sslmode=disable" up

## db/migrations/new name=$1: create a new database migration
.PHONY: db/migrations/new
db/migrations/new:
	@echo "Creating migration files for ${name}..."
	docker run --rm -v $$(pwd)/migrations:/migrations migrate/migrate create -seq -ext=.sql -dir=/migrations ${name}

## db/migrations/up: apply all up migrations
.PHONY: db/migrations/up
db/migrations/up:
	@echo "Running all up migrations..."
	docker compose run --rm migrate up

## db/migrations/down: rollback last migration
.PHONY: db/migrations/down
db/migrations/down:
	@echo "Rolling back last migration..."
	docker compose run --rm migrate down 1

## db/migrations/reset: reset database (run all down then all up)
.PHONY: db/migrations/reset
db/migrations/reset:
	@echo "Resetting database..."
	docker compose run --rm migrate down && docker compose run --rm migrate up

## db/migrations/force version=$1: force database version
.PHONY: db/migrations/force
db/migrations/force:
	@echo "Forcing version to ${version}..."
	docker compose run --rm migrate force ${version}

## db/psql: connect to PostgreSQL using psql
.PHONY: db/psql
db/psql:
	docker exec -it postgres-db psql -U user -d mydb

## db/setup: setup database with migrations and seed data
.PHONY: db/setup
db/setup:
	@echo "Setting up database..."
	@$(MAKE) db/migrations/up
	@$(MAKE) db/seed
	@echo "Database setup complete!"

## build: build the application
.PHONY: build
build:
	@echo "Building application..."
	docker compose build

## up: start all services
.PHONY: up
up:
	@echo "Starting all services..."
	docker compose up -d

## down: stop all services
.PHONY: down
down:
	@echo "Stopping all services..."
	docker compose down

## logs: show application logs
.PHONY: logs
logs:
	docker compose logs -f go

## =========================
## TEST TARGETS
## =========================

## test: run all tests
.PHONY: test
test:
	@echo "Running all tests..."
	docker compose exec go go test -v ./...

## test/models: run only model tests
.PHONY: test/models
test/models:
	@echo "Running model tests..."
	docker compose exec go go test -v ./internal/models/...

## test/handlers: run only handler tests
.PHONY: test/handlers
test/handlers:
	@echo "Running handler tests..."
	docker compose exec go go test -v ./internal/handlers/...

## test/unit: run unit tests (models and handlers)
.PHONY: test/unit
test/unit:
	@echo "Running unit tests..."
	docker compose exec go go test -v ./internal/models/... ./internal/handlers/...

## test/coverage: run tests with coverage report
.PHONY: test/coverage
test/coverage:
	@echo "Running tests with coverage..."
	docker compose exec go sh -c 'go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html'
	@echo "Coverage report generated: coverage.html"

## test/coverage/text: run tests with text coverage report
.PHONY: test/coverage/text
test/coverage/text:
	@echo "Running tests with text coverage..."
	docker compose exec go go test -cover ./...

## test/short: run tests in short mode
.PHONY: test/short
test/short:
	@echo "Running tests in short mode..."
	docker compose exec go go test -short ./...

## test/race: run tests with race detector
.PHONY: test/race
test/race:
	@echo "Running tests with race detector..."
	docker compose exec go go test -race ./...

## test/bench: run benchmark tests
.PHONY: test/bench
test/bench:
	@echo "Running benchmark tests..."
	docker compose exec go go test -bench=. -benchmem ./...

## test/specific name=$1: run specific test by name pattern
.PHONY: test/specific
test/specific:
	@echo "Running tests matching pattern: ${name}"
	docker compose exec go go test -v -run ${name} ./...

## test/assets: run asset-related tests
.PHONY: test/assets
test/assets:
	@echo "Running asset-related tests..."
	docker compose exec go go test -v -run "Asset" ./...

## test/clean: clean test cache
.PHONY: test/clean
test/clean:
	@echo "Cleaning test cache..."
	docker compose exec go go clean -testcache

## test/deps: install test dependencies
.PHONY: test/deps
test/deps:
	@echo "Installing test dependencies..."
	docker compose exec go sh -c 'go get github.com/stretchr/testify@v1.8.4 && go get github.com/DATA-DOG/go-sqlmock@v1.5.0'
	@echo "Test dependencies installed"

## test/setup: setup test environment (dependencies + database)
.PHONY: test/setup
test/setup:
	@echo "Setting up test environment..."
	@$(MAKE) test/deps
	@$(MAKE) db/setup
	@echo "Test environment ready!"

## test/integration: run integration tests (requires running services)
.PHONY: test/integration
test/integration:
	@echo "Running integration tests..."
	docker compose exec go go test -v -tags=integration ./internal/integration/...

## test/watch: run tests on file changes (requires entr installed in container)
.PHONY: test/watch
test/watch:
	@echo "Watching for file changes and running tests..."
	docker compose exec go sh -c 'find . -name "*.go" -not -path "./vendor/*" | entr -r go test ./internal/models/...'

## test/verbose: run tests with verbose output
.PHONY: test/verbose
test/verbose:
	@echo "Running tests with verbose output..."
	docker compose exec go go test -v ./...

## test/ci: run tests for CI environment (with coverage and race detector)
.PHONY: test/ci
test/ci:
	@echo "Running CI test suite..."
	docker compose exec go go test -race -cover -coverprofile=coverage.out -covermode=atomic ./...

## =========================
## DEVELOPMENT WORKFLOWS
## =========================

## dev/test: development test workflow (run tests and show coverage)
.PHONY: dev/test
dev/test:
	@$(MAKE) test/unit
	@$(MAKE) test/coverage/text

## dev/full-test: full test suite for development
.PHONY: dev/full-test
dev/full-test:
	@echo "Running full test suite..."
	@$(MAKE) test/clean
	@$(MAKE) test/unit
	@$(MAKE) test/race
	@$(MAKE) test/coverage

## clean: remove containers and volumes
.PHONY: clean
clean:
	@echo "Cleaning up..."
	docker compose down -v
	@echo "Clean complete!"

## help: show available commands
.PHONY: help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Database:"
	@echo "  db/setup              - Setup database with migrations and seed data"
	@echo "  db/seed               - Generate and update seed data"
	@echo "  db/psql               - Connect to PostgreSQL"
	@echo "  db/migrations/up      - Apply all up migrations"
	@echo "  db/migrations/down    - Rollback last migration"
	@echo "  db/migrations/new     - Create new migration (name=NAME)"
	@echo ""
	@echo "Testing:"
	@echo "  test                  - Run all tests"
	@echo "  test/models           - Run only model tests"
	@echo "  test/handlers         - Run only handler tests"
	@echo "  test/unit             - Run unit tests (models + handlers)"
	@echo "  test/coverage         - Run tests with HTML coverage report"
	@echo "  test/coverage/text    - Run tests with text coverage report"
	@echo "  test/race             - Run tests with race detector"
	@echo "  test/bench            - Run benchmark tests"
	@echo "  test/specific         - Run specific test (name=PATTERN)"
	@echo "  test/assets           - Run asset-related tests"
	@echo "  test/ci               - Run CI test suite"
	@echo "  test/setup            - Setup test environment"
	@echo ""
	@echo "Development:"
	@echo "  dev/test              - Development test workflow"
	@echo "  dev/full-test         - Full test suite"
	@echo ""
	@echo "Application:"
	@echo "  build                 - Build application"
	@echo "  up                    - Start all services"
	@echo "  down                  - Stop all services"
	@echo "  run/api               - Run application"
	@echo "  logs                  - Show application logs"
	@echo "  clean                 - Remove containers and volumes"