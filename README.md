# Password Manager

Open-source, self-hosted enterprise password manager: companies, folders, roles, fine-grained access grants, end-to-end encryption, browser autofill and shared environment variables for developers.

Status: early development. The back end is being built phase by phase.

- Requirements: [docs/requirements.md](docs/requirements.md)
- Back-end plan: [docs/backend-implementation-plan.md](docs/backend-implementation-plan.md)
- Contributor and agent rules: [CLAUDE.md](CLAUDE.md)

## Repository layout

| Path | What |
| --- | --- |
| `server/` | Go API (Gin, PostgreSQL), clean architecture |
| `web/` | React web app (Tailwind, shadcn/ui) |
| `extension/` | Chrome and Firefox extension |
| `spec/` | `openapi.yaml`, the API contract shared by all clients |
| `deploy/` | Docker Compose files and `.env.example` |
| `docs/` | Requirements, plans, design notes |

## Back-end quick start

Needs Docker and Go 1.26.

Everything runs from `server/`:

| Command | What it does |
| --- | --- |
| `make up` | Starts PostgreSQL, Mailpit, Prometheus and Grafana in Docker. No server. |
| `make dev` | `make up`, then runs the server on your machine with `go run`. No image is built. Ctrl+C stops it. |
| `make dev-watch` | Same as `make dev`, but rebuilds and restarts the server when you save a `.go` or `.sql` file. |
| `make start` | Builds the production server image and runs everything in Docker, then runs a smoke check against it. |
| `make down` | Stops everything. Data stays in Docker volumes. |

While the stack is up:

| URL | What |
| --- | --- |
| http://localhost:8080/healthz | API health check |
| http://localhost:8080/api/v1/meta | API version |
| http://localhost:3000 | Grafana: server dashboard (view freely; edit as `admin` / `pm-dev-only`) |
| http://localhost:9091 | Prometheus |
| http://localhost:9090/metrics | Raw server metrics (internal port) |
| http://localhost:8025 | Mailpit: every email the server sends |

Other commands (`make help` lists them all):

| Command | What |
| --- | --- |
| `make test` | All tests, including PostgreSQL and SMTP integration tests |
| `make lint` | golangci-lint, including the clean-architecture layer rules |
| `make generate` | Regenerate sqlc queries and the OpenAPI server code |
| `make check` | Module tidiness, lint, vet and test: what CI runs |
| `make build` | Build the production image |
| `make logs` | Follow the server container's logs (after `make start`) |

Configuration is by environment variables, all listed in [deploy/.env.example](deploy/.env.example).
