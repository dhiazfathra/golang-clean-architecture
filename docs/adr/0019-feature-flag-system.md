# ADR-0019: Feature flag system with 3-tier cache

## Status
Accepted

## Context
As the platform grows, the ability to toggle features at runtime — without redeployment — becomes
essential for safe rollouts, A/B experimentation, and incident mitigation (kill-switching a
problematic feature). The project needs a feature flag mechanism that is:

1. **Fast on the hot path** — flag checks happen on every request when gating routes; they must
   not add meaningful latency.
2. **Consistent across instances** — in a multi-instance deployment, toggling a flag must
   propagate to all processes within a bounded window.
3. **Durable** — flag state must survive process restarts and infrastructure failures.
4. **Admin-manageable** — operators must be able to create, toggle, and delete flags via an API
   without touching code or configuration files.

Three caching strategies were evaluated:

1. **Database-only** — every `IsEnabled` call queries Postgres. Simple, but adds a network
   round-trip per request and puts unnecessary load on the database for a read-heavy, rarely
   changing dataset.

2. **In-process cache only** (`sync.Map` with background refresh) — zero-latency reads, but
   in a multi-instance deployment, toggling a flag only takes effect after the next refresh
   cycle on each instance. No shared state between instances.

3. **3-tier hybrid: sync.Map → Valkey → Postgres** — the in-process `sync.Map` serves the hot
   path with zero allocation. On a miss, Valkey (already deployed for session storage) provides
   a shared cache layer. Postgres remains the source of truth, queried only on a cold start or
   double cache miss. Writes update all three layers synchronously (Postgres first), and a
   background refresh goroutine keeps instances consistent within the configured interval.

The project already uses Valkey for session storage (ADR-0005), so the 3-tier approach adds no
new infrastructure dependency. The `sync.Map` type provides lock-free reads, which is ideal for
a read-heavy, write-rare workload like feature flags.

## Decision

1. **Package** — Implement feature flags as a platform package at `pkg/platform/featureflag`,
   following the same layout as other platform concerns (RBAC, session, observability).

2. **3-tier cache** — Use the hybrid architecture:
   `sync.Map (in-process) → Valkey (shared cache) → Postgres (source of truth)`.
   - `sync.Map` provides zero-alloc, lock-free reads on the hot path.
   - Valkey keys are namespaced with `ff:` prefix and have a TTL of 2× the refresh interval.
   - Postgres is the authoritative store; all writes go to Postgres first.

3. **Background refresh** — A goroutine reloads all flags from Postgres into Valkey and the
   in-process cache at a configurable interval (`Config.FeatureFlagRefreshTTL`, default 30s,
   env var `FEATURE_FLAG_REFRESH_TTL`). This bounds the staleness window across instances.

4. **Write-through** — On Create, Toggle, or Delete, all three cache layers are updated
   synchronously (Postgres → Valkey → sync.Map). The calling instance sees the change
   immediately; other instances converge within one refresh interval.

5. **Route middleware** — `featureflag.RequireFlag(svc, "key")` is an Echo middleware that
   returns `404 Not Found` when the flag is disabled, making gated features completely
   invisible to clients rather than returning a `403` that would leak feature existence.

6. **Admin API** — CRUD endpoints under `/admin/feature-flags`, gated by
   `featureflag:manage` RBAC permission. Follows the same admin route pattern as audit
   endpoints (ADR-0010, M11).

7. **Data model** — The `feature_flags` table includes a `metadata JSONB` column for future
   extensibility (targeting rules, rollout percentages) without requiring schema migrations.
   Soft delete via `is_deleted` flag follows the project convention (ADR-0012).

8. **Valkey client reuse** — The existing `valkey.Client` instance created for session storage
   is reused for feature flag caching. No additional Valkey connection is required.

## Consequences

**Easier:**
- Feature toggles can be flipped at runtime via a single API call without redeployment,
  enabling safe rollouts and instant kill-switches.
- The `RequireFlag` middleware makes it trivial to gate any route behind a feature flag with
  a single line of code in route registration.
- The in-process cache ensures zero additional latency on the hot path for flag checks.
- The `metadata` JSONB column provides a natural extension point for percentage-based rollouts,
  user-segment targeting, or A/B experiment configuration — without schema changes.
- No new infrastructure dependency: Valkey and Postgres are already in the stack.

**Harder:**
- Flag state can be stale for up to one refresh interval (default 30s) on instances that did
  not perform the write. This is an accepted trade-off for the simplicity of the polling model.
- The background refresh goroutine must be started explicitly (`StartRefresh`) and its context
  must be cancelled on shutdown to avoid goroutine leaks.
- Operators must understand the 3-tier model to reason about propagation delays when toggling
  flags in a multi-instance deployment.

**Deferred:**
- Percentage-based rollouts and user-segment targeting can be implemented by interpreting the
  `metadata` JSONB field — the schema is already in place.
- Event-driven cache invalidation (Postgres LISTEN/NOTIFY or Valkey Pub/Sub) could replace
  polling for near-instant propagation if the refresh interval proves too slow.
- A UI dashboard for flag management can be built on top of the existing admin API endpoints.
- Audit logging of flag changes (who toggled what, when) can be added by emitting events to
  the event store, following the project's event-sourcing pattern.
