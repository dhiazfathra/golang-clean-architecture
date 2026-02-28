# ADR-0008: Timestamp-prefixed migration filenames

## Status
Accepted

## Context
Database migration tools (golang-migrate, goose, flyway) support sequential numbering
(`001_`, `002_`) or timestamp prefixes for ordering migrations. Sequential numbers cause
merge conflicts when two developers create migrations in parallel branches — both pick the
next available number. Timestamps eliminate this conflict because each developer's timestamp
is unique.

## Decision
All migration files use the format `YYYYMMDDHHMMSS_<descriptive_name>.up.sql` and
`YYYYMMDDHHMMSS_<descriptive_name>.down.sql`. The timestamp is the UTC wall clock time at
the moment the migration file is created. golang-migrate processes files in lexicographic
order, which matches chronological order for this format.

## Consequences
**Easier:**
- No renumbering conflicts when merging parallel branches.
- Filename encodes creation time, making the migration history self-documenting.
- Standard approach supported natively by golang-migrate's file source.

**Harder:**
- Timestamp must be captured at creation time; clock skew between developer machines is
  theoretically possible but in practice irrelevant for coarse (second) granularity.

**Deferred:**
- None. This is a simple naming convention with no future complications.
