# ADR-0016: CLI module generator with text/template scaffolding

## Status
Accepted

## Context
Adding a new domain module requires creating ~9 files with consistent boilerplate:
handler, service interface, service implementation, repository interface, repository
implementation, events, state, read-model, and route registration snippet. Writing
these by hand is tedious and introduces inconsistencies. Code generation from templates
is the standard solution; the question is which tool to use.

## Decision
Use Go's standard `text/template` package. Templates live in `templates/module/*.tmpl`.
The generator CLI (`cmd/generate/main.go`) accepts `-module=<Name>` and optional
`-fields=<field:type,...>` flags, renders each template with the module name (and
derived forms: lower, pascal, snake), writes files to `pkg/module/<name>/`, and prints
a wiring snippet for `cmd/server/main.go`.

No external codegen tool (protoc, sqlc, ent) is introduced. No AST manipulation.

## Consequences
**Easier:**
- One command produces a fully wired, compilable CRUD module skeleton.
- Templates are plain `.tmpl` text files — any developer can read and modify them.
- No external tool installation required beyond the Go toolchain.
- The generated code follows all project conventions automatically (BaseReadModel,
  constructor DI, event types, etc.).

**Harder:**
- Template syntax (`{{.PascalName}}`, `{{.Fields}}`) must be maintained as conventions
  evolve — templates and shared context must stay in sync.
- The generator writes files but does not modify existing files (e.g., it cannot
  automatically insert route registration into `main.go`); the wiring snippet must be
  added manually.

**Deferred:**
- Interactive mode (prompted field entry instead of flags).
- Template versioning — if templates diverge from conventions, a `generate --upgrade`
  command could re-scaffold existing modules.
