# ADR-0015: Idempotent database seeders operating through event store

## Status
Accepted

## Context
On first boot (and in CI), the database needs a super-admin user, default roles with
appropriate permissions, and per-module seed users. These seeds must be safe to run
multiple times (idempotent) so that a re-run of `make seed` does not create duplicates
or fail with unique constraint violations. Since the system uses event sourcing, seeds
should create real events — not bypass the event store with direct SQL inserts — so that
the audit trail is complete from day one.

## Decision
`pkg/platform/seeder` provides a `Seed(ctx, store, rbacSvc, authSvc, modules)` function.
For each seed item it:
1. Checks whether the entity already exists via a read-model query.
2. If not, calls the appropriate service method (which appends events to the store).
3. Logs the outcome.

The RBAC module registry (`rbac.RegisterModule(name, permissions)`) drives automatic role
and permission seeding: each registered module's seed role and user are created if absent.
`make seed` calls `cmd/seed/main.go` which wires and invokes `seeder.Seed(...)`.

## Consequences
**Easier:**
- `make seed` is safe to run at any time, in any environment.
- New modules automatically get their seed role/user by calling `rbac.RegisterModule`.
- Seeds are auditable: the event store shows that the super-admin was created by the
  seeder at time T.

**Harder:**
- Seeder depends on multiple services (auth, RBAC), so its wiring in `main.go` / seed
  binary is more involved than a simple SQL script.
- If a service method changes signature, the seeder must be updated in lockstep.

**Deferred:**
- Environment-specific seed data (staging fixtures, demo data) — current seeder only
  handles the minimal bootstrap set required for the application to function.
