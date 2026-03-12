# ADR-0021: Structured Logging with zerolog

## Status
Accepted

## Context
Go's standard `log` package provides no structured output, making it difficult to query
or filter logs in production observability tools (Datadog, Loki, CloudWatch). The newer
`log/slog` standard library package addresses this but carries measurable allocation
overhead per log call. High-throughput service paths — request handling, event processing,
background workers — emit logs on every operation, so logger performance is on the critical
path. Three viable structured-logging libraries exist: `slog` (stdlib), `zap` (Uber), and
`zerolog`. All three support JSON output and log levels; they differ in API ergonomics,
allocation behaviour, and ecosystem maturity.

## Decision
Use `rs/zerolog` as the sole logging library. A single package-level `zerolog.Logger` is
initialised in `main.go` and injected via constructors into components that need it.
All log calls use the chained builder API (`log.Info().Str("key", val).Msg("…")`).
Direct use of `fmt.Println` or the stdlib `log` package is prohibited outside of test
helpers.

## Consequences
**Easier:**
- Zero heap allocations on the hot path — zerolog is designed around this guarantee.
- JSON output works out of the box; switching to pretty-print for local dev requires one
  `zerolog.ConsoleWriter` line in `main.go`.
- Chained builder API makes it impossible to emit a log line without a message, reducing
  inconsistent log shapes across the codebase.
- Log level filtering (`zerolog.SetGlobalLevel`) is trivially configurable at startup via
  an environment variable.

**Harder:**
- The chained API is unfamiliar to developers used to `slog`'s key-value variadic style;
  forgetting to call `.Msg()` or `.Send()` silently drops the log entry.
- `zerolog.Logger` is a value type, not an interface — passing it through constructors
  adds a small but visible type dependency; mocking requires wrapping in a custom interface
  if log output must be asserted in tests.
- Structured context (request IDs, trace IDs) must be manually threaded via
  `logger.With().Str(…).Logger()` or stored on `context.Context`; there is no automatic
  propagation.

**Deferred:**
- If OpenTelemetry log bridge support becomes necessary, a `slog` adapter can be layered
  on top of zerolog without changing call sites — only the initialisation in `main.go`
  changes.
