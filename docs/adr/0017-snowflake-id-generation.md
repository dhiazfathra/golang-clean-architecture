# ADR-0017: Snowflake IDs for all database primary keys

## Status
Accepted

## Context
Three competing options exist for generating primary keys in a distributed Go service:

1. **Database `SERIAL` / `BIGSERIAL`** — simple, but requires a round-trip to the DB to
   obtain the ID before the application can use it. Fails silently in a sharded or
   multi-writer topology because sequences are per-node.

2. **UUID v4** — globally unique with no coordination, but 128-bit random values cause
   random insertion order in B-tree indexes. This leads to page splits, index
   fragmentation, and higher write amplitudes on every `INSERT`.

3. **Time-ordered distributed ID (Snowflake)** — 64-bit integer composed of a timestamp,
   machine/node identifier, and a sequence counter. IDs are monotonically increasing
   within a node, preserving B-tree locality. No DB round-trip required. The algorithm
   was originally designed by Twitter and has several maintained Go implementations.

This project prioritises write performance, auditability (IDs encode creation time), and
operational simplicity (single binary, no external ID service).

## Decision
Use the Snowflake algorithm (`github.com/bwmarrin/snowflake`) for all entity primary keys.

- **Type in Go:** `int64` (the library's `snowflake.ID` underlying type).
- **Column type in PostgreSQL:** `BIGINT NOT NULL`.
- **Node ID:** read from the `SNOWFLAKE_NODE_ID` environment variable; defaults to `1`
  when the variable is absent. Each replica **must** be assigned a unique node ID
  (0–1023) to guarantee global uniqueness.
- **Initialisation:** a single `*snowflake.Node` is created once at startup in
  `pkg/platform/database` and injected wherever IDs are generated — no global variable.
- **No `SERIAL` or `UUID` columns** for entity PKs. Lookup / join tables that have no
  independent identity may still use composite keys.

## Consequences
**Easier:**
- IDs are monotonically increasing per node, preserving B-tree index locality and
  eliminating page-split write amplification.
- The creation timestamp is embedded in the ID; no extra query is needed to determine
  approximate record age for debugging or data pipelines.
- ID generation is purely in-process — no DB round-trip, no coordination service.
- 64-bit integers are smaller on the wire and in indexes than 128-bit UUIDs.

**Harder:**
- Each running replica requires a distinct `SNOWFLAKE_NODE_ID` (0–1023). This is an
  operational concern: node IDs must be assigned and not reused within the same
  millisecond window.
- IDs are not opaque — the timestamp component is trivially extractable, which leaks
  approximate record-creation time to anyone who can read the ID. Acceptable for this
  project; note for security reviews.
- The 64-bit space (≈ 2^63 positive values) is smaller than UUID v4, though at
  4096 IDs/ms/node it would take ~73 years to exhaust, which is acceptable.

**Deferred:**
- If the project ever shards beyond 1024 nodes, the node ID bit-width must be revisited.
- UUID v7 (time-ordered, standardised in RFC 9562) could replace Snowflake if broader
  ecosystem tooling support becomes relevant.
