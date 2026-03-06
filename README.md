# Go Clean Architecture

[![Go Version](https://img.shields.io/github/go-mod/go-version/dhiazfathra/golang-clean-architecture)](https://go.dev/)
[![Build Status](https://github.com/dhiazfathra/golang-clean-architecture/actions/workflows/ci.yml/badge.svg)](https://github.com/dhiazfathra/golang-clean-architecture/actions)
[![codecov](https://codecov.io/gh/dhiazfathra/golang-clean-architecture/graph/badge.svg?token=5R78WXtIbj)](https://codecov.io/gh/dhiazfathra/golang-clean-architecture)
[![Go Report Card](https://goreportcard.com/badge/github.com/dhiazfathra/golang-clean-architecture)](https://goreportcard.com/report/github.com/dhiazfathra/golang-clean-architecture)
[![License](https://img.shields.io/github/license/dhiazfathra/golang-clean-architecture)](LICENSE)
[![Release](https://img.shields.io/github/v/release/dhiazfathra/golang-clean-architecture)](https://github.com/dhiazfathra/golang-clean-architecture/releases)

A Go backend demonstrating a **modular monolith** with **event sourcing**, **RBAC with field-level permissions**, and **Datadog observability**.

---

## Overview

| Concern | Approach |
|---------|----------|
| State persistence | Event sourcing — append-only events, CQRS read models |
| HTTP | Echo v4 |
| Database | PostgreSQL via sqlx (no ORM) |
| Session auth | Valkey (Redis-compatible) cookie sessions |
| RBAC | Module + action + field-level permissions; event-sourced role store |
| Primary keys | Snowflake `int64` (no UUID, no SERIAL) |
| Observability | Datadog APM, logs, metrics, profiler, DBM, error tracking |
| API docs | Scalar UI (`/docs`) from embedded OpenAPI spec |

---

## Full Setup (One Command)

### Prerequisites

Install Go, Docker, Docker Compose, psql, Make, and git:

```bash
# macOS / Linux / WSL2
bash scripts/setup-prerequisites.sh
```

To check if prerequisites are already installed:

```bash
bash scripts/setup-prerequisites.sh --check
```

### Bootstrap & Run

```bash
make setup
```

This will: start Postgres + Valkey, apply migrations, seed initial data, and start the server.

| Flag | Description |
|------|-------------|
| `--skip-prereqs` | Skip prerequisite check |
| `--no-seed` | Skip database seeding |
| `--no-run` | Setup infrastructure only |
| `--reset` | Tear down existing infra before starting |

```bash
# Example: reset and rebuild everything
make setup-reset

# Example: setup infra only, don't start server
bash scripts/setup.sh --no-run
```

### Windows (WSL2)

1. Install WSL2: `wsl --install` from PowerShell (admin)
2. Open the WSL2 terminal (Ubuntu)
3. Clone the repo and run: `bash scripts/setup-prerequisites.sh`
4. Run: `make setup`

> Docker Desktop must be installed with WSL2 integration enabled.

---

## Quick Start (Manual)

```bash
# 1. Start infrastructure (Postgres + Valkey)
make infra-up

# 2. Apply database migrations
DATABASE_URL=postgres://app:app@localhost:5432/app?sslmode=disable make migrate

# 3. Seed initial roles and users
SEED_SUPER_ADMIN_PASSWORD=secret123 \
SEED_DEFAULT_MODULE_PASSWORD=module123 \
DATABASE_URL=postgres://app:app@localhost:5432/app?sslmode=disable \
VALKEY_URL=valkey://localhost:6379 \
make seed

# 4. Run the server
DATABASE_URL=postgres://app:app@localhost:5432/app?sslmode=disable \
VALKEY_URL=valkey://localhost:6379 \
make run
```

The server listens on `:8080` by default. API docs: [http://localhost:8080/docs](http://localhost:8080/docs)

---

## Architecture

### Event Sourcing Write Path

```
HTTP request
  → Echo middleware (tracing, metrics)
  → Session middleware (RequireSession)
  → RBAC middleware (RequirePermission)
  → Handler
    → Service.Command(...)
      → Aggregate.Apply(event)
      → EventStore.Append(events)
```

### CQRS Read Path (async projection)

```
ProjectionRunner (poll loop, 200ms)
  → EventStore.LoadUnprojected(projectorID)
    → Projector.Project(event)  [UPSERT into read-model table]
  → Response: Handler → rbac.FilterResponse → JSON
```

### Session Auth

Every protected route is guarded by `session.RequireSession`, which reads a signed session cookie, validates it against Valkey, and stores `userID` in the Echo context. Handlers retrieve it via `session.UserID(c)`.

### RBAC

Roles and permissions are event-sourced (`RoleCreated`, `PermissionGranted`, `RoleAssigned`). Each route carries `rbac.RequirePermission(svc, module, action)` middleware. GET handlers call `rbac.FilterResponse` to strip fields the caller's roles don't permit.

See [docs/rbac.md](docs/rbac.md) for the full permission model.

---

## Module Creation

### Option A — Generator (recommended)

```bash
make generate module=widget fields="label:string,weight:float64"
# Paste the printed wiring snippet into cmd/server/main.go
make migrate
make seed
```

### Option B — Manual (5 steps)

```
1. Create pkg/module/<name>/ with 9 files:
   model.go  projections.go  projector.go
   repository.go  repository_pg.go
   service.go  handler.go  routes.go  register.go

2. Migration: YYYYMMDDHHMMSS_<name>_read.up.sql
   (must include 5 audit columns + is_deleted)

3. Wire 4 lines in cmd/server/main.go

4. register.go init():
   eventstore.Register[...]  +  rbac.RegisterModule(...)

5. make seed
```

See [docs/new-module-checklist.md](docs/new-module-checklist.md) for the full checklist.

---

## API Endpoints

All API endpoints are prefixed with `/api/v1`. Health probes remain at root for infrastructure compatibility.

### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/login` | Authenticate; sets session cookie |
| POST | `/api/v1/auth/logout` | Invalidate session |
| GET | `/api/v1/auth/me` | Current user info |

### Users

| Method | Path | Auth | Permission |
|--------|------|------|------------|
| POST | `/api/v1/users` | session | `user:create` |
| GET | `/api/v1/users` | session | `user:list` |
| GET | `/api/v1/users/:id` | session | `user:read` |
| DELETE | `/api/v1/users/:id` | session | `user:delete` |
| GET | `/api/v1/admin/users/:id` | session | `user:read` (includes soft-deleted) |

### Orders

| Method | Path | Auth | Permission |
|--------|------|------|------------|
| POST | `/api/v1/orders` | session | `order:create` |
| GET | `/api/v1/orders` | session | `order:list` |
| GET | `/api/v1/orders/:id` | session | `order:read` |
| DELETE | `/api/v1/orders/:id` | session | `order:delete` |

### RBAC (admin)

| Method | Path | Permission |
|--------|------|------------|
| POST | `/api/v1/admin/roles` | `rbac:manage` |
| GET | `/api/v1/admin/roles` | `rbac:manage` |
| GET | `/api/v1/admin/roles/:id` | `rbac:manage` |
| DELETE | `/api/v1/admin/roles/:id` | `rbac:manage` |
| POST | `/api/v1/admin/roles/:id/permissions` | `rbac:manage` |
| DELETE | `/api/v1/admin/roles/:id/permissions/:perm` | `rbac:manage` |
| GET | `/api/v1/admin/users/:id/roles` | `rbac:manage` |

### Feature Flags (admin)

| Method | Path | Permission |
|--------|------|------------|
| GET | `/api/v1/admin/feature-flags` | `featureflag:manage` |
| POST | `/api/v1/admin/feature-flags` | `featureflag:manage` |
| PATCH | `/api/v1/admin/feature-flags/:key` | `featureflag:manage` |
| DELETE | `/api/v1/admin/feature-flags/:key` | `featureflag:manage` |

Feature flags use a hybrid 3-tier cache: **sync.Map (in-process) → Valkey (shared) → Postgres (source of truth)**. Use `featureflag.RequireFlag(svc, "key")` middleware to gate any route behind a flag.

### Environment Variables

| Method | Path | Permission |
|--------|------|------------|
| POST | `/api/v1/envs` | `envvar:manage` |
| GET | `/api/v1/envs/:platform/:key` | `envvar:manage` |
| GET | `/api/v1/envs/:platform` | `envvar:manage` |
| PUT | `/api/v1/envs/:platform/:key` | `envvar:manage` |
| DELETE | `/api/v1/envs/:platform/:key` | `envvar:manage` |

Dynamic environment variables scoped by platform (`mobile`, `web`, `be`, etc.). Uses the same hybrid 3-tier cache as feature flags: **sync.Map (in-process) → Valkey (shared) → Postgres (source of truth)**.

### Audit

| Method | Path | Permission |
|--------|------|------------|
| GET | `/api/v1/admin/audit/:type/:id` | `audit:read` |

### Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness probe |
| GET | `/health/ready` | Readiness probe (Postgres + Valkey) |

### Docs (non-production only)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/docs` | Scalar API UI |
| GET | `/openapi.yaml` | Raw OpenAPI 3.1 spec |

---

## Configuration

All values can be set via environment variables or a YAML file (`CONFIG_FILE=path/to/config.yaml`).

| Env Var | Default | Description |
|---------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL DSN (`postgres://user:pass@host/db?sslmode=disable`) |
| `VALKEY_URL` | — | Valkey/Redis URL (`valkey://host:6379`) |
| `ENV` | `development` | Runtime environment (`development` \| `production`) |
| `LISTEN_ADDR` | `:8080` | TCP address to listen on |
| `SEED_SUPER_ADMIN_PASSWORD` | — | Password for `admin@system.local` |
| `SEED_DEFAULT_MODULE_PASSWORD` | — | Password for module admin users |
| `SNOWFLAKE_NODE_ID` | `1` | Snowflake node ID (1–1023) |
| `FEATURE_FLAG_REFRESH_TTL` | `30s` | Feature flag cache refresh interval (min `1s`) |
| `ENV_VAR_REFRESH_TTL` | `30s` | Dynamic env var cache refresh interval (min `1s`) |
| `DD_API_KEY` | — | Datadog API key (optional; disables APM if unset) |
| `DD_ENV` | — | Datadog environment tag |
| `DD_SERVICE` | — | Datadog service name |
| `DD_VERSION` | — | Datadog version tag |

---

## Testing

```bash
# Run all unit tests
make test

# Run with coverage report
make cover

# Static analysis
make vet

# Lint
make lint
```

Tests use hand-rolled mocks (struct with function fields) and `testutil.NewMockDB` for sqlx+sqlmock. No external services are required for unit tests — `observability.InitNoop()` is called automatically from `testutil.init()`.

---

## Observability

When `DD_API_KEY` is set, the following data flows to Datadog:

| Signal | What you see |
|--------|--------------|
| **APM** | Distributed traces: HTTP span → session span → DB span; service map |
| **Logs** | Structured JSON with `dd.trace_id` on every line (log–trace correlation) |
| **Metrics** | `golang-clean-arch.http.request.count` tagged by method/route/status |
| **Profiler** | CPU + heap profiles (always-on, low overhead) |
| **DBM** | SQL queries linked back to APM spans |
| **Error Tracking** | 5xx responses captured with stack trace |

In development (`ENV=development`) the tracer runs in no-op mode — no agent required.

---

## ADRs

Architectural decisions are documented in [docs/adr/README.md](docs/adr/README.md).
