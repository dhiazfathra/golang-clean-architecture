# ADR-0012: Soft-delete via `is_deleted` flag with automatic CRUD filtering

## Status
Accepted

## Context
Hard-deleting rows from read-model tables destroys information that may be needed for audits,
reporting, or recovery. On the other hand, requiring every query to include
`WHERE is_deleted = false` is error-prone and easy to forget. We need a mechanism that
makes deleted records invisible to normal queries without requiring per-query discipline.

## Decision
Every read-model table has an `is_deleted BOOLEAN NOT NULL DEFAULT false` column
(provided by `BaseReadModel`, ADR-0010). The generic CRUD helpers in
`pkg/platform/database/crud` automatically append `AND is_deleted = false` to all
`Get[T]` and `Select[T]` calls. A separate `GetIncludingDeleted[T]` variant exists for
administrative use cases where deleted records must be visible. "Deleting" a record
means publishing a `*Deleted` event whose projector sets `is_deleted = true` in the
read-model row — the event and the row both persist.

## Consequences
**Easier:**
- Normal queries are safe by default; no per-query delete filter required.
- Deleted records are recoverable: publish a `*Restored` event to set `is_deleted = false`.
- The full history of a deleted entity remains in the event store (ADR-0001).
- `is_deleted` column aligned with `BaseReadModel` means it is present on every table
  automatically.

**Harder:**
- Read-model tables grow indefinitely because rows are never physically removed.
  Archival / purge jobs must be written separately if table size becomes a concern.
- Unique constraints must account for soft-deleted rows (e.g., unique email must consider
  only non-deleted rows, or use a partial unique index).

**Deferred:**
- Periodic hard-delete of rows that have been soft-deleted for >N days (data retention policy).
- Partial unique indexes for soft-deleted uniqueness constraints.
