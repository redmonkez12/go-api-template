# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

| Command | Description |
|---------|-------------|
| `make run` | Run the API server |
| `make build` | Compile binary to `bin/api` |
| `make test` | Run tests with race detector and coverage |
| `make docker-up` / `make docker-down` | Start/stop local infrastructure (Postgres, Redis, Adminer, Loki, Alloy, Grafana) |
| `make migrate-up` / `make migrate-down` | Run/rollback database migrations |
| `make migrate-create NAME=xxx` | Create new migration files |
| `make swagger` | Generate Swagger docs from annotations |
| `make install-tools` | Install `migrate` and `swag` CLI tools |
| `make deps` | Download and tidy Go modules |

Run a single test: `go test -v -race ./internal/auth/ -run TestFunctionName`

## Architecture

**Layered structure with manual dependency injection.** All wiring happens in `cmd/api/main.go`.

```
Handler (HTTP request/response) → Service (business logic) → Repository (database queries)
```

Each domain lives in its own package under `internal/`:
- **auth** — PASETO token auth, Argon2id password hashing, refresh tokens in Redis, login/register/verify/reset handlers, auth middleware
- **user** — User model, Bun ORM repository (CRUD, queries by email/ID/verification token)
- **config** — Loads from env vars with `.env` fallback
- **database** — Bun ORM model definitions
- **email** — SMTP service for verification and password reset emails
- **http** — Chi router setup, security headers middleware, HTTP server
- **httputil** — JSON response helpers and error code constants
- **logging** — slog-based structured logger, request logging middleware with context injection
- **ratelimit** — Redis-based rate limiting (per-IP and per-email cooldowns)

**Key tech choices:** Chi v5 router, Bun ORM (PostgreSQL), PASETO v4 tokens, Redis for refresh tokens and rate limits, `log/slog` for structured logging.

## Adding a New Domain/Feature

1. Create package under `internal/` with model, repository, service, and handler
2. Add database model to `internal/database/models.go`
3. Create migration: `make migrate-create NAME=create_xxx_table`
4. Wire dependencies in `cmd/api/main.go`
5. Register routes in `internal/http/router.go`
6. Add Swagger annotations to handler methods, then `make swagger`

## Key Patterns

- **Request/response types** are defined in handler files alongside their handlers
- **Custom error types** per service (e.g., `auth.ErrInvalidCredentials`, `user.ErrNotFound`, `user.ErrDuplicateEmail`)
- **Auth middleware** (`authMiddleware.RequireAuth`) extracts PASETO from `Authorization: Bearer <token>` header
- **Cookie vs JSON auth responses** — auto-detected via `Origin` header (browser gets HttpOnly cookies, API clients get JSON)
- **Swagger UI** only available when `APP_ENV=dev`
- **Middleware order matters** — CORS → security headers → recoverer → request ID → real IP → request logger → compression
- **DB model mapping** — database models (`internal/database`) are separate from domain models; mapped via functions like `mapDBUserToModel()`

## Environment

Copy `.env.example` to `.env`. Key variables: `SERVER_PORT`, `APP_ENV` (dev/prod), `DB_*`, `REDIS_*`, `PASETO_KEY` (32-byte hex), `SMTP_*`, `FRONTEND_URL`, `TRUSTED_ORIGINS` (CORS).
