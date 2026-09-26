# url-shortener
A basic URL shortening service

This is a hobby project to learn Golang. The application has 2 components:
1. Admin Service: a RESTful API for creating accounts and managing short URLs.
2. Url Redirector: a small HTTP server that redirects short URLs to their original long URLs and records clicks.

## Project Structure
```
url-shortener/
├── cmd/
│   ├── admin/          # Admin API entry point
│   ├── redirect/       # Redirect service entry point
│   ├── migrate/        # Applies database migrations (run before the services)
│   ├── seed/           # Creates a user directly, e.g. when registration is off
│   └── healthcheck/    # Container health probe (the images have no shell)
├── internal/
│   ├── config/         # Typed, validated configuration per service
│   ├── platform/       # Postgres and Redis connections
│   ├── auth/           # JWT issuing/verification and auth middleware
│   ├── user/           # Accounts: model, repository, service
│   ├── shortlink/      # Short links: model, repository, service
│   ├── analytics/      # Click recording
│   ├── cache/          # Redis link cache and rate-limit storage
│   ├── ratelimit/      # Request limits
│   ├── httpapi/        # HTTP handlers for the admin API and the redirect service
│   └── testdb/         # Helpers for integration tests
├── migrations/         # Versioned SQL schema (goose), embedded in the binaries
└── docs/               # Generated Swagger docs
```
`cmd/*` only wires dependencies together. Services depend on repository
interfaces, so they can be tested with the in-memory implementations.

## Getting Started
### Prerequisites
- Go 1.25 or later (CI and the Docker images build with Go 1.27)
- Postgres 15 or later
- Redis 7.4 or later
- Docker and Docker Compose (optional)

### Configuration
Copy `.env.example` to `.env` and fill it in. Outside production
(`ENV` other than `production`) the services read `.env` if present; real
environment variables always take precedence. Each service validates its
configuration at startup and lists every missing or invalid value at once.

Required for both services: `SHORT_URL_BASE`, `DB_HOST`, `DB_USERNAME`,
`DB_DATABASE`, `REDIS_HOST`. The admin API also requires `JWT_SIGNING_KEY`
(at least 32 bytes) and `CORS_ORIGINS` (explicit origins; `*` is rejected).
See `.env.example` for every option and its default.

### Run locally
1. Create the database, then apply migrations:
   ```bash
   createdb urlshortener
   go run ./cmd/migrate           # or: go run ./cmd/migrate status
   ```
   The services refuse to start while migrations are pending.
2. Start the services:
   ```bash
   go run ./cmd/admin
   go run ./cmd/redirect
   ```

### Run with Docker
```bash
cp .env.example .env   # set JWT_SIGNING_KEY and REDIS_PASSWORD
docker compose up --build
```
Compose starts Postgres and Redis, runs the migrations once, then starts the
admin API on 8086 and the redirector on 8085. Postgres and Redis are on an
internal network with no published ports, as in production: only the two Go
services can reach them.

One `Dockerfile` builds both images (`--target admin` or `--target redirect`).
They are distroless, run as a non-root user, contain static stripped binaries,
and have a built-in health check against `/readyz`. The admin image also
contains `migrate` and `seed`.

### Deploying
Postgres and Redis are expected on a private network that only the Go services
can reach. On such a network `DB_SSLMODE=disable` is acceptable if the database
does not offer TLS; otherwise keep the production default, `require`. Keep a
Redis password regardless.

Run `migrate` as a release step before starting new versions of the services
(with the admin image: `/app/migrate`).
Services only need read/write access to the tables, so they can use a
database role without DDL rights; only `migrate` needs to alter the schema.

Operations:
- **Health:** both services serve `GET /healthz` (process is up) and
  `GET /readyz` (Postgres and Redis answer within a second; 503 otherwise).
- **Shutdown:** on SIGTERM or SIGINT the services stop accepting requests, finish
  in-flight ones, write any queued clicks, then close connections (up to 15s).
- **Logs:** structured, one JSON object per line in production
  (`LOG_FORMAT`), with an access-log entry and `X-Request-Id` for every request.
- **Redis:** set `maxmemory` and `maxmemory-policy volatile-lru`. Every key the
  app writes has a TTL; the services log a warning at startup if Redis has no
  memory limit.
- **Postgres:** each process opens at most `DB_MAX_OPEN_CONNS` connections.

Clicks are written in batches off the request path. If the database falls
behind and the in-memory queue (10,000 clicks) fills, further clicks are
dropped and counted in the logs rather than slowing redirects.

### Try it
- Register with `POST /api/v1/users/register`:
  ```json
  { "email": "alice@example.com", "name": "Alice", "password": "at-least-8-chars" }
  ```
  The email is the login ID. It must be unique and is stored lowercase, so matching is case-insensitive.
- Log in with `POST /api/v1/users/login` using `{"email": "...", "password": "..."}` to get a token.
- Send the token as `Authorization: Bearer <token>`.
- `GET /api/v1/urls` returns `{"items": [...], "next_cursor": "..."}`; pass `next_cursor` back as `cursor` for the next page (it is `null` on the last page).
- If registration is disabled (`REGISTRATION_ENABLED=false`), create users with:
  ```bash
  SEED_PASSWORD='at-least-8-chars' go run ./cmd/seed -email alice@example.com -name Alice
  ```

## Tests
```bash
go test ./...
```
CI (GitHub Actions) runs on every pull request and push to `main`:
- **Lint:** gofmt, `go vet`, staticcheck, a tidy `go.mod`, and generated Swagger docs.
- **Tests:** unit and integration tests with the race detector, against Postgres and Redis service containers.
- **Vulnerabilities:** govulncheck.
- **Compose smoke test:** builds the images, starts the stack, and checks registration, link creation and redirects. It also checks that Postgres and Redis are not reachable from the host, that the containers run as non-root, and that clicks are written on shutdown.

Dependabot proposes weekly updates for Go modules, base images and Actions.
Integration tests for the Postgres and Redis code run when `SHORTENER_TEST_DB`
names a local database they may modify (Redis DB 13 by default, or
`SHORTENER_TEST_REDIS_DB`):
```bash
createdb urlshortener_test
SHORTENER_TEST_DB=urlshortener_test go test -p 1 ./...
```

## API Documentation
Swagger UI is served at `/api/v1/swagger/index.html` on the admin API when
`ENABLE_SWAGGER` is on. Regenerate the docs after changing handler annotations:
```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g main.go \
  -d ./cmd/admin,./internal/httpapi/admin,./internal/shortlink,./internal/user -o docs
```
