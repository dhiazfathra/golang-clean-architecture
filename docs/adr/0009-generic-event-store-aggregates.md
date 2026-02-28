# ADR-0009: Generic event store / aggregate root via Go generics

## Status
Accepted

## Context
Each event-sourced module (User, Order, Role) requires the same boilerplate: an aggregate
struct that holds state, a slice of uncommitted events, a version counter, and Load/Save
methods that replay events through an apply function. Without generics, this boilerplate
must be copy-pasted and adapted per module, creating maintenance burden and divergence risk.
Go 1.18 introduced generics, making a shared, type-safe aggregate root possible.

## Decision
`pkg/platform/eventstore` provides `Aggregate[T any]` where `T` is the domain state struct.
Each module provides three things:
1. A state struct (e.g., `UserState`).
2. Event types implementing the `Event` interface.
3. An `applyFn func(*T, Event)` that mutates state given an event.

The generic aggregate handles uncommitted event tracking, version incrementing, snapshot
serialisation, and Load/Save via the `Store` interface. Modules never write Load/Save logic.

## Consequences
**Easier:**
- Zero Load/Save boilerplate per module — three items and the module is event-sourced.
- Consistent event replay, versioning, and snapshot behaviour across all modules.
- Type-safe: `aggregate.State` is `T`, not `any`; no type assertions in module code.

**Harder:**
- Go generics add compile complexity; tooling support (IDE, linters) was initially rough
  but is mature as of Go 1.22.
- The apply function must handle all event types via a type switch — no automatic dispatch.

**Deferred:**
- Optimistic concurrency control (version conflict detection on Save) — the Store interface
  is designed to accept it as a future option.
