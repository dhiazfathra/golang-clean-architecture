# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the Go Clean Architecture project.
Each ADR documents a significant architectural decision: its context, the decision made, and its consequences.

## How to Read ADRs
ADRs are numbered and written using the template:
- **Context** — why the decision was needed
- **Decision** — what was decided
- **Consequences** — what becomes easier, harder, or is deferred

## Index

| # | Title | Status | File |
|---|-------|--------|------|
| 0001 | Use event sourcing for state persistence | Accepted | [0001-use-event-sourcing.md](0001-use-event-sourcing.md) |
| 0002 | Modular monolith layout (no microservices) | Accepted | [0002-modular-monolith-layout.md](0002-modular-monolith-layout.md) |
| 0003 | Echo as HTTP router | Accepted | [0003-echo-http-router.md](0003-echo-http-router.md) |
| 0004 | sqlx over ORM for database access | Accepted | [0004-sqlx-over-orm.md](0004-sqlx-over-orm.md) |
| 0005 | Valkey (Redis-compatible) for session storage | Accepted | [0005-valkey-session-store.md](0005-valkey-session-store.md) |
| 0006 | Poll-based projection runner over CDC/message bus | Accepted | [0006-poll-based-projection-runner.md](0006-poll-based-projection-runner.md) |
| 0007 | Constructor-based DI with no framework | Accepted | [0007-constructor-di-no-framework.md](0007-constructor-di-no-framework.md) |
| 0008 | Timestamp-prefixed migration filenames | Accepted | [0008-timestamp-migration-filenames.md](0008-timestamp-migration-filenames.md) |
| 0009 | Generic event store / aggregate root via Go generics | Accepted | [0009-generic-event-store-aggregates.md](0009-generic-event-store-aggregates.md) |
| 0010 | BaseReadModel with mandatory audit fields derived from events | Accepted | [0010-base-read-model-audit-fields.md](0010-base-read-model-audit-fields.md) |
| 0011 | Datadog observability injected at platform layer only | Accepted | [0011-datadog-observability-platform-layer.md](0011-datadog-observability-platform-layer.md) |
| 0012 | Soft-delete via `is_deleted` flag with automatic CRUD filtering | Accepted | [0012-soft-delete-via-is-deleted-flag.md](0012-soft-delete-via-is-deleted-flag.md) |
| 0013 | RBAC with module-level and field-level permissions | Accepted | [0013-rbac-module-and-field-level.md](0013-rbac-module-and-field-level.md) |
| 0014 | Field-level permission filtering via generic JSON response filter | Accepted | [0014-field-level-permission-filtering.md](0014-field-level-permission-filtering.md) |
| 0015 | Idempotent database seeders operating through event store | Accepted | [0015-idempotent-database-seeders.md](0015-idempotent-database-seeders.md) |
| 0016 | CLI module generator with text/template scaffolding | Accepted | [0016-cli-module-generator.md](0016-cli-module-generator.md) |
| 0017 | Snowflake IDs for all database primary keys | Accepted | [0017-snowflake-id-generation.md](0017-snowflake-id-generation.md) |
| 0018 | Scalar as the API documentation renderer | Accepted | [0018-scalar-api-docs.md](0018-scalar-api-docs.md) |
| 0019 | Feature flag system with 3-tier cache | Accepted | [0019-feature-flag-system.md](0019-feature-flag-system.md) |

## How to Add a New ADR

1. Copy the template below into `docs/adr/NNNN-<short-title>.md`.
2. Fill in Context, Decision, and Consequences.
3. Add a row to the index table above.

```markdown
# ADR-NNNN: <Title>

## Status
Accepted

## Context
<Why is this decision needed? What forces are at play?>

## Decision
<What was decided?>

## Consequences
<What becomes easier? What becomes harder? What is deferred?>
```
