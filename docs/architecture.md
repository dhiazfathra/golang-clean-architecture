# Architecture

## Dependency Graph

```
cmd/server/main.go (DI wiring)
  │
  ├── pkg/platform/config
  ├── pkg/platform/observability   ← Datadog APM / logs / metrics / profiler
  ├── pkg/platform/database        ← sqlx connection + traced driver
  ├── pkg/platform/eventstore      ← event store, aggregate root, projection runner
  ├── pkg/platform/session         ← Valkey store, RequireSession middleware
  ├── pkg/platform/rbac            ← service, middleware, handler, routes
  ├── pkg/platform/seeder          ← idempotent bootstrap via event store
  ├── pkg/platform/health          ← liveness + readiness handlers
  ├── pkg/platform/docs            ← Scalar UI + OpenAPI spec handler
  │
  ├── pkg/module/auth              ← login / logout / me
  ├── pkg/module/user              ← user CRUD + admin routes
  └── pkg/module/order             ← order CRUD

Modules never import each other or platform/observability directly.
Cross-module interfaces are defined by the consuming module and
satisfied by an adapter in main.go.
```

---

## Event Flow

```
HTTP request
  │
  ▼
Echo middleware chain
  ├── observability.EchoMiddleware  (APM trace start, Datadog span)
  ├── observability.RequestMetrics  (statsd counter increment)
  └── session.RequireSession        (cookie → Valkey → userID in context)
  │
  ▼
RBAC middleware: rbac.RequirePermission(svc, module, action)
  │  loads roles for session.UserID(c), checks permissions
  │
  ▼
Handler
  │  reads request body, path params, calls service
  │
  ▼
Service.Command(ctx, cmd)
  │  loads aggregate from event store (replay)
  │  applies domain logic → new event(s)
  │
  ▼
EventStore.Append(ctx, aggregateID, events, expectedVersion)
  │  INSERTs to events table (Postgres, optimistic concurrency)
  │
  ▼
Handler returns 2xx  ──────────────────────────────────────────────┐
                                                                    │
ProjectionRunner (goroutine, 200ms poll)                           │
  │  SELECT events not yet seen by this projector                  │
  │                                                                │
  ▼                                                                │
Projector.Project(event)                                          │
  │  UPSERT into read-model table                                 │
  │                                                                │
  ▼                                                                │
Read model is up-to-date ──────────────────────────────────────────┘

GET request → Handler → repo.Get / repo.List
                      → rbac.FilterResponse(c, result)
                      → JSON response (fields filtered by caller's roles)
```

---

## Data Flow

### Write Path

```
Command (HTTP body)
  → Service validates + builds event struct
  → Aggregate.Apply(event) updates in-memory state
  → EventStore.Append persists to Postgres `events` table
     columns: id, aggregate_id, aggregate_type, event_type,
              version, payload (JSONB), metadata (JSONB), created_at
```

### Read Path

```
events table
  → ProjectionRunner polls every 200ms
  → Projector.Project(event) runs domain-specific UPSERT
     into <module>_read table
     (populates created_at, created_by, updated_at, updated_by from event metadata)
  → GET handler queries <module>_read table via sqlx
  → rbac.FilterResponse strips disallowed fields
  → JSON response
```

---

## RBAC Permission Model

Every permission is a triple: `{module, action, FieldPolicy}`.

```
Role "user_admin"
  └── Permission { module: "user", action: "read",
                   fields: { mode: "deny", fields: ["password_hash"] } }
  └── Permission { module: "user", action: "list",
                   fields: { mode: "all" } }
  └── Permission { module: "user", action: "create",
                   fields: { mode: "all" } }

Role "super_admin"
  └── Permission { module: "*", action: "*",
                   fields: { mode: "all" } }   ← wildcard
```

**Field policy evaluation** — when the caller has multiple roles:
- Collect all permissions matching the requested `{module, action}`.
- Merge field policies: `all` beats `allow`; `deny` is additive.
- `rbac.FilterResponse` marshals the response to a `map[string]any`, removes denied
  or non-allowed fields, then re-marshals to JSON.

---

## Audit Field Derivation

```
Event.Metadata["user_id"]   ──▶  created_by / updated_by
Event.Timestamp()           ──▶  created_at / updated_at
```

Each projector UPSERT sets:
- `created_by` / `created_at` only on the INSERT path (conflict target = aggregate PK).
- `updated_by` / `updated_at` on every UPSERT.

The read model exposes these via `database.BaseReadModel` (embedded struct).

---

## Soft Delete

```
Service.Delete(ctx, id)
  → Aggregate.Apply(DeletedEvent{UserID: actorID, ...})
  → EventStore.Append

ProjectionRunner
  → Projector.Project(DeletedEvent)
    → UPDATE <table> SET is_deleted = true, updated_by = ..., updated_at = ...
      WHERE id = $1

crud.Get[T](db, "SELECT ... WHERE id=$1 AND is_deleted = false", id)
  → returns sql.ErrNoRows when soft-deleted  → handler returns 404

crud.GetIncludingDeleted[T](db, ...)   ← admin endpoints only
  → returns the row regardless of is_deleted flag
```
