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

## test: run tests
.PHONY: test
test:
	@echo "Running tests..."
	docker compose exec go go test ./...

## clean: remove containers and volumes
.PHONY: clean
clean:
	@echo "Cleaning up..."
	docker compose down -v
	@echo "Clean complete!"