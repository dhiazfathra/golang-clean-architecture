# ADR-0004: sqlx over ORM for database access

## Status
Accepted

## Context
ORM libraries (GORM, ent) provide conveniences like automatic migrations and association
handling, but they abstract SQL in ways that obscure query behaviour, make performance
tuning harder, and generate surprising N+1 queries. Raw `database/sql` gives full control
but requires manual struct scanning. A middle ground is needed: write SQL explicitly, get
struct scanning for free.

## Decision
Use `github.com/jmoiron/sqlx` which wraps `database/sql` with struct scanning, named
parameters, and `In` query helpers — without hiding the SQL. All queries are written as
explicit SQL strings. Struct tags (`db:"column_name"`) map result columns to fields.
No query builder or code generator is used.

## Consequences
**Easier:**
- Full control over every SQL query; no ORM "magic" to debug.
- `sqlx.NamedExec` and `sqlx.Select` eliminate manual `Scan` loops.
- Performance tuning is straightforward — the SQL is right there.
- No N+1 by accident; joins must be written explicitly.

**Harder:**
- More SQL to write per repository; no auto-generated CRUD.
- Schema changes require updating both migration files and struct tags manually.

**Deferred:**
- Query builder (e.g., squirrel) could be introduced for dynamic filter construction
  if query complexity grows beyond manageable SQL strings.
