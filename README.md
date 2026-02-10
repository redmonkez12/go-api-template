# Go API Template

A production-ready Go REST API template with batteries included: authentication, email verification, observability, and more.

## What's Included

- **Authentication**: PASETO v4.local tokens, refresh token rotation, Argon2id password hashing
- **Email**: Verification emails, password reset flow (SMTP)
- **Database**: PostgreSQL with Bun ORM, migrations
- **Cache**: Redis for refresh tokens, rate limiting, password reset tokens
- **Router**: Chi with structured middleware (CORS, security headers, request logging)
- **Observability**: Loki + Grafana + Alloy for log aggregation
- **API Docs**: Swagger UI (dev mode only)
- **Docker**: Multi-stage Dockerfile, docker-compose for local dev
- **CI/CD**: GitHub Actions workflow for building and pushing Docker images

## Quick Start

```bash
# 1. Clone and configure
cp .env.example .env
# Edit .env with your settings (especially PASETO_KEY)

# 2. Start infrastructure
make docker-up

# 3. Install tools (first time only)
make install-tools

# 4. Run migrations
make migrate-up

# 5. Generate Swagger docs
make swagger

# 6. Start the API
make run
```

The API will be available at `http://localhost:8080`.
Swagger UI at `http://localhost:8080/swagger/index.html` (dev mode only).

## Project Structure

```
.
├── cmd/api/              # Application entrypoint
│   └── main.go           # Server bootstrap, DI wiring
├── internal/
│   ├── auth/             # Authentication (handlers, service, PASETO, middleware)
│   ├── config/           # Environment-based configuration
│   ├── database/         # Bun ORM models and helpers
│   ├── email/            # SMTP email service with HTML templates
│   ├── http/             # Router, server, security middleware
│   ├── httputil/         # Response helpers, error codes
│   ├── logging/          # Structured logger (slog) + request logging middleware
│   ├── ratelimit/        # Redis-based rate limiting
│   └── user/             # User domain (model, repository)
├── migrations/           # SQL migration files
├── config/               # Observability configs (Loki, Grafana, Alloy)
├── docs/                 # Generated Swagger documentation
└── .github/workflows/    # CI/CD pipelines
```

## Available Make Targets

| Command | Description |
|---------|-------------|
| `make run` | Run the application |
| `make build` | Build binary to `bin/api` |
| `make test` | Run tests with coverage |
| `make docker-up` | Start PostgreSQL, Redis, Adminer, and observability stack |
| `make docker-down` | Stop all containers |
| `make docker-logs` | Tail container logs |
| `make migrate-up` | Run database migrations |
| `make migrate-down` | Rollback last migration |
| `make migrate-create NAME=x` | Create a new migration |
| `make swagger` | Generate Swagger documentation |
| `make deps` | Download and tidy dependencies |
| `make install-tools` | Install migrate and swag CLI tools |
| `make docker-build` | Build production Docker image |
| `make docker-run` | Run production container |

## Adding a New Feature

1. Create a new package under `internal/` (e.g., `internal/todo/`)
2. Add model, repository, service, and handler files
3. Wire dependencies in `cmd/api/main.go`
4. Add routes in `internal/http/router.go`
5. Create migrations with `make migrate-create NAME=create_todos_table`

## Auth Flow

1. **Register** (`POST /auth/register`) - Creates user, sends verification email
2. **Verify Email** (`GET /auth/verify-email?token=...`) - Activates account
3. **Login** (`POST /auth/login`) - Returns PASETO access token + refresh token
4. **Refresh** (`POST /auth/refresh`) - Rotates tokens (old refresh token revoked)
5. **Logout** (`POST /auth/logout`) - Revokes refresh token, clears cookies
6. **Forgot Password** (`POST /auth/forgot-password`) - Sends reset email
7. **Reset Password** (`POST /auth/reset-password`) - Updates password with token

Tokens are returned as JSON for API clients, or as HttpOnly cookies for browser clients (detected via `Origin` header).

## Observability

The template includes a full observability stack for development:

- **Grafana** at `http://localhost:3000` - Log visualization and dashboards
- **Loki** at `http://localhost:3100` - Log aggregation backend
- **Alloy** - Collects Docker container logs and ships to Loki

All API request logs are structured JSON, automatically parsed and indexed by Loki.
