# ADR-0002: Modular monolith layout (no microservices)

## Status
Accepted

## Context
Domain isolation is valuable — it enforces bounded contexts and prevents tight coupling.
Microservices achieve isolation but introduce significant operational overhead: distributed
tracing, inter-service networking, independent deployment pipelines, and the complexity of
distributed transactions. For the scale and team size of this project, that overhead is
premature. We want domain separation without the ops burden.

## Decision
Ship a single binary. Domain modules live under `pkg/module/<name>/`. Each module owns its
own handler, service, repository, and event types. Modules communicate exclusively through
Go interfaces defined in their own package and satisfied by adapters wired in `cmd/server/main.go`.
No module may import another module's package directly.

## Consequences
**Easier:**
- Single deploy artifact; no service mesh or inter-service auth needed.
- Local development: one `go run` starts everything.
- Transactions span modules trivially (shared DB connection).
- Refactoring module boundaries is a code change, not an infrastructure change.

**Harder:**
- Independent scaling of a hot module requires extracting it to a service later.
- Discipline required to enforce "no cross-module imports" — enforced via Go module linting.

**Deferred:**
- If a module needs independent scaling, it can be extracted to a microservice; the interface
  boundary (ADR-0007) makes this a well-defined seam.
