# New Module Checklist

## Option A — Generator (recommended)

```bash
# 1. Generate scaffolding
go run cmd/generate/main.go -module=<name> -fields="<field>:<type>,..."
# Example:
go run cmd/generate/main.go -module=widget -fields="label:string,weight:float64"

# 2. Copy the wiring snippet printed to stdout into cmd/server/main.go

# 3. Apply the generated migration
make migrate

# 4. Seed roles for the new module
make seed
```

The generator creates all 9 files in `pkg/module/<name>/` plus the migration.

---

## Option B — Manual (9 files)

### Step 1 — Create the module directory

```
pkg/module/<name>/
  model.go           ← domain model + event type constants
  projections.go     ← read-model struct (embed database.BaseReadModel)
  projector.go       ← event → UPSERT logic
  repository.go      ← interface (Get, List)
  repository_pg.go   ← sqlx implementation
  service.go         ← command methods (Create, Delete, …)
  handler.go         ← Echo handler funcs
  routes.go          ← RegisterRoutes func
  register.go        ← init(): Register events + RBAC module
```

### Step 2 — Migration

Filename: `migrations/YYYYMMDDHHMMSS_<name>_read.up.sql`

Required columns (non-negotiable):
```sql
CREATE TABLE IF NOT EXISTS <name>_read (
    id          BIGINT PRIMARY KEY,
    -- domain columns …
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  TEXT        NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by  TEXT        NOT NULL DEFAULT '',
    is_deleted  BOOLEAN     NOT NULL DEFAULT FALSE
);
```

### Step 3 — Wire into cmd/server/main.go

Add 4 lines (projector registration + service + handler + routes):
```go
<name>Projector := <name>.NewProjector(db)
<name>ReadRepo  := <name>.NewPgReadRepository(db)
<name>Svc       := <name>.NewService(es, <name>ReadRepo)

runner.Register(<name>Projector)
<name>.RegisterRoutes(protected, <name>.NewHandler(<name>Svc), rbacSvc)
```

### Step 4 — register.go init()

```go
func init() {
    eventstore.Register[<Name>State]("<name>", applyFn)
    rbac.RegisterModule("<name>", []string{"create", "read", "list", "delete"})
}
```

### Step 5 — Seed

```bash
make seed
# Creates <name>_admin role with all actions + an <name>_admin user
```

---

## Rules That Must Never Be Broken

| Rule | Why |
|------|-----|
| Every read-model struct embeds `database.BaseReadModel` | Consistent audit fields on every response |
| Every read-model table has the 5 audit columns + `is_deleted` | Projector writes to these columns; `crud.Get` filters by `is_deleted` |
| Every protected route has `rbac.RequirePermission(rbacSvc, module, action)` | Unauthenticated / unauthorised requests are rejected before the handler runs |
| Every GET handler calls `rbac.FilterResponse(c, result)` | Field-level permissions are enforced; callers only see fields they are allowed |
| No module imports `pkg/platform/observability`, `pkg/platform/session`, or other modules | Observability and cross-module dependencies are injected via `main.go` adapters |
| IDs are Snowflake `int64`, stored as `BIGINT` | Consistent PK strategy; no UUID / SERIAL |
| Migration filenames use `YYYYMMDDHHMMSS` wall-clock prefix | Avoids collisions; `make migrate` applies them in sort order |
