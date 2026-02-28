# ADR-0007: Constructor-based DI with no framework

## Status
Accepted

## Context
Dependency injection frameworks (Wire, Dig, Fx) reduce boilerplate but add indirection: the
dependency graph is implicit, compile errors become runtime panics, and debugging wiring
problems requires understanding the framework's internals. For a project of this size, the
wiring code is manageable and explicit dependencies are a feature, not a bug.

## Decision
All dependencies are passed as constructor arguments. Every package exposes a `New*(...)`
constructor function. Wiring happens exclusively in `cmd/server/main.go` and
`cmd/generate/main.go`. No global service locators, no `init()`-registered singletons
(except for event type and RBAC module registration which are idempotent).

## Consequences
**Easier:**
- The dependency graph is immediately visible by reading `main.go`.
- Compile-time errors for missing or wrong-typed dependencies — no runtime surprises.
- Unit testing is trivial: pass mock implementations directly to constructors.
- No framework to learn, upgrade, or debug.

**Harder:**
- `main.go` grows verbose as the number of modules increases.
- Adding a new transitive dependency requires threading it through multiple constructors.

**Deferred:**
- If `main.go` becomes unmanageable (>500 lines of wiring), a code-generated DI approach
  (Wire) can be adopted without changing module code — only `main.go` changes.
