# ADR-0001: Use event sourcing for state persistence

## Status
Accepted

## Context
The application requires complete audit trails for all state changes. Implementing audit logging
as a separate concern (triggers, interceptors, change-data-capture) is error-prone and often
incomplete. We need a model where the history of every entity is a first-class citizen, not an
afterthought. Additionally, replaying events to rebuild state supports temporal queries and
debugging production issues without additional instrumentation.

## Decision
Every state change is represented as an immutable, append-only event stored in the `events`
table. The canonical source of truth is the event stream, not the current row value.
Read models are projections computed from events and stored in separate tables for query
performance. Aggregates (in `pkg/platform/eventstore`) load their state by replaying events
since the last snapshot.

## Consequences
**Easier:**
- Free, complete audit log with no extra per-module code.
- Temporal queries: replay to any point in time.
- Debugging: reproduce any past state deterministically.
- Decoupled write (events) and read (projections) optimisation paths.

**Harder:**
- Reads require projection tables to be kept up-to-date (poll-based runner, ADR-0006).
- Large aggregates accumulate many events; snapshots are needed (handled by event store).
- Developers unfamiliar with event sourcing have a steeper initial learning curve.

**Deferred:**
- Snapshot compaction strategy (beyond a configurable threshold).
- Event schema versioning / upcasting for breaking event changes.
