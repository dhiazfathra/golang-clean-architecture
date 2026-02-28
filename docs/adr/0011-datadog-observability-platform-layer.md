# ADR-0011: Datadog observability injected at platform layer only

## Status
Accepted

## Context
Observability (tracing, metrics, logging correlation) is cross-cutting. The naive approach
is to import `dd-trace-go` in every module that needs a span or a metric. This creates
hundreds of Datadog import sites, couples domain logic to a specific vendor, and makes
testing harder (every test must initialise a tracer). A cleaner approach keeps observability
at the infrastructure boundary.

## Decision
All Datadog initialisation, middleware registration, and client wrapping lives exclusively
in `pkg/platform/observability`. Domain modules (`pkg/module/*`) never import this package
or any `dd-trace-go` package. Tracing is automatic because:
- HTTP spans are created by the `ddecho` middleware registered in `main.go`.
- DB spans are created by the `traced sqlx` wrapper returned by `observability.NewDB(...)`.
- Valkey spans are created by the traced client wrapper in `pkg/platform/session`.

Tests use `observability.InitNoop()` which registers a no-op tracer and no-op statsd client.

## Consequences
**Easier:**
- Zero Datadog imports in any module — modules are vendor-agnostic.
- Swapping Datadog for another APM requires changes only in `pkg/platform/observability`.
- Tests run without a Datadog agent; `InitNoop()` is a single call in test setup.
- Automatic tracing without per-handler span creation code.

**Harder:**
- Custom spans inside business logic are not directly possible from module code. If a module
  needs a custom span, it must accept a `context.Context` and use the stdlib `context`
  package — the platform layer adds the span by convention in the traced wrappers.

**Deferred:**
- Custom span creation API for modules (e.g., a `telemetry.Span(ctx, name)` helper that
  modules can import without coupling to Datadog directly).
