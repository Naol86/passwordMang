# Back-end developer commands. Run `make help` for the list.
#
#   make up       PostgreSQL, Mailpit, Prometheus, Grafana in Docker
#   make dev      make up + the server on your machine with `go run` (no image build)
#   make start    everything in Docker, including the production server image, then a smoke check
#   make down     stop everything

COMPOSE        := docker compose -f ../deploy/docker-compose.dev.yml
COMPOSE_ALL    := $(COMPOSE) --profile app
SQLC_IMAGE     := sqlc/sqlc:1.31.1
LINT_IMAGE     := golangci/golangci-lint:v2.14.0
AIR            := github.com/air-verse/air@v1.67.4
DOCKER_RUN     := docker run --rm -v "$(CURDIR)":/src -w /src

# Settings for a server running on the host against the Docker services (all dev-only values).
DEV_ENV := PM_ENV=development PM_LOG_LEVEL=debug \
	PM_HTTP_ADDR=:8080 PM_METRICS_ADDR=:9090 \
	PM_DATABASE_URL="postgres://pm:pm-dev-only@localhost:5433/pm?sslmode=disable" \
	PM_SMTP_HOST=localhost PM_SMTP_PORT=1025 PM_SMTP_TLS=none \
	PM_SMTP_FROM="Password Manager <vault@pm.localhost>"

TEST_DB_URL    ?= postgres://pm:pm-dev-only@localhost:5433/pm?sslmode=disable
TEST_ENV       := PM_TEST_DATABASE_URL="$(TEST_DB_URL)" PM_TEST_SMTP_HOST=localhost PM_TEST_MAILPIT_URL=http://localhost:8025

.PHONY: help up dev dev-watch start down logs test test-unit lint tidy-check generate generate-check check build

help: ## Show this help
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-15s %s\n", $$1, $$2}'

up: ## Start PostgreSQL, Mailpit, Prometheus and Grafana in Docker (not the server)
	$(COMPOSE) up -d --wait
	@echo "Grafana  http://localhost:3000   Prometheus http://localhost:9091   Mail http://localhost:8025"

# A server container from `make start` would hold the same ports, so it is stopped first.
dev: up ## Run the server on your machine with go run (Ctrl+C to stop)
	@$(COMPOSE_ALL) stop server >/dev/null 2>&1 || true
	$(DEV_ENV) go run ./cmd/server

dev-watch: up ## Like dev, but rebuilds and restarts the server when you save a file
	@$(COMPOSE_ALL) stop server >/dev/null 2>&1 || true
	$(DEV_ENV) go run $(AIR) -c .air.toml

start: ## Build and run everything in Docker, including the production server image, then smoke-check it
	$(COMPOSE_ALL) up --build -d --wait
	@./scripts/smoke-check.sh || { $(COMPOSE_ALL) logs --tail 50 server; exit 1; }
	@echo "API http://localhost:8080   Grafana http://localhost:3000   Mail http://localhost:8025"

down: ## Stop everything (data is kept in Docker volumes)
	$(COMPOSE_ALL) down

logs: ## Follow the logs of the server container started by make start
	$(COMPOSE_ALL) logs -f server

test: up ## Run all tests, including database and SMTP tests, with the race detector
	$(TEST_ENV) go test -race -count=1 ./...

test-unit: ## Run tests that need no database
	go test -race -count=1 ./...

lint: ## Run golangci-lint (pinned version, in Docker)
	$(DOCKER_RUN) -v pm-golangci-cache:/root/.cache $(LINT_IMAGE) golangci-lint run ./...

tidy-check: ## Fail if go.mod or go.sum need `go mod tidy`
	go mod tidy -diff

generate: ## Regenerate sqlc queries and the OpenAPI server code
	$(DOCKER_RUN) $(SQLC_IMAGE) generate
	go tool oapi-codegen -config oapi-codegen.yaml ../spec/openapi.yaml

generate-check: generate ## Fail if generated code is out of date
	git diff --exit-code -- internal/adapter/postgres/sqlcgen internal/adapter/httpapi/apigen

check: tidy-check lint test ## Everything CI runs: module tidiness, lint and all tests
	go vet ./...

build: ## Build the production Docker image
	docker build --target runtime -t pm-server:local .
