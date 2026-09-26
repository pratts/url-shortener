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
│   └── seed/           # Creates a user directly, e.g. when registration is off
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
- Go 1.24.1 or later
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
admin API on 8086 and the redirector on 8085.

### Deploying
Run `migrate` as a release step before starting new versions of the services.
Services only need read/write access to the tables, so they can use a
database role without DDL rights; only `migrate` needs to alter the schema.

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
