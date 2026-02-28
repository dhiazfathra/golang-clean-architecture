# ADR-0006: Poll-based projection runner over CDC/message bus

## Status
Accepted

## Context
Event sourcing requires projections — read models built from the event stream. Options include:
Change-Data-Capture (CDC) tooling (Debezium), a message bus (Kafka, NATS), or polling the
events table directly. CDC and message buses are powerful but add significant infrastructure
complexity. For the current scale, consistency lag of ~500ms is acceptable and the simpler
approach should be preferred.

## Decision
Each projector is a goroutine that polls the `events` table every 500ms for events it has
not yet processed. The last processed event ID (cursor) is stored in a `projection_cursors`
table keyed by projector name. The projection runner (`pkg/platform/projection`) manages
goroutine lifecycle and cursor persistence. No external message bus is required.

## Consequences
**Easier:**
- No additional infrastructure (no Kafka, no Debezium, no separate message broker).
- Projectors are simple functions: `func(event Event) error`.
- Easy to add new projectors: register with the runner, no broker topic configuration.
- Cursors in the DB mean projectors restart safely after crash/restart.

**Harder:**
- ~500ms eventual consistency between write and read model update.
- Under high write load, the poller may fall behind; backpressure handling is manual.

**Deferred:**
- Replacing polling with PostgreSQL `LISTEN/NOTIFY` would reduce lag to near-zero with no
  module changes (the projector interface remains the same).
- Message bus integration (for cross-service event publishing) is a future seam.
