# ADR-0010: BaseReadModel with mandatory audit fields derived from events

## Status
Accepted

## Context
Every read-model table needs the same five audit columns: `created_at`, `created_by`,
`updated_at`, `updated_by`, and `is_deleted`. Without a shared convention, individual
modules define these inconsistently, forget some fields, or use different column names.
Since state changes arrive as events and events carry metadata (actor, timestamp), these
fields can always be derived from the event stream without additional inputs.

## Decision
`pkg/platform/database` exports `BaseReadModel`:
```go
type BaseReadModel struct {
    CreatedAt time.Time `db:"created_at" json:"created_at"`
    CreatedBy string    `db:"created_by" json:"created_by"`
    UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
    UpdatedBy string    `db:"updated_by" json:"updated_by"`
    IsDeleted bool      `db:"is_deleted" json:"-"`
}
```
Every domain read-model struct embeds `BaseReadModel`. Projectors populate these fields
using the `AuditFieldsFromEvent(e Event) BaseReadModel` helper, which extracts actor from
`e.Metadata()["actor"]` and timestamp from `e.Timestamp()`.

## Consequences
**Easier:**
- Zero per-module audit field code — embed and call `AuditFieldsFromEvent`.
- Consistent column names across all tables; tooling and queries are predictable.
- `is_deleted` alignment with ADR-0012 (soft-delete) is automatic.
- `json:"-"` on `IsDeleted` prevents accidental exposure in API responses.

**Harder:**
- Projectors must ensure event metadata always contains the `actor` key; missing metadata
  results in empty `created_by`/`updated_by` (not a hard failure, but a data quality issue).

**Deferred:**
- Additional standard fields (e.g., `tenant_id`) can be added to `BaseReadModel` later
  without changing existing projectors — they would simply remain zero-valued until
  populated.
