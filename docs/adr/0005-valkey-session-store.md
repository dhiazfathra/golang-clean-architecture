# ADR-0005: Valkey (Redis-compatible) for session storage

## Status
Accepted

## Context
HTTP sessions need to survive server restarts and support horizontal scaling. Storing sessions
in-process memory fails both requirements. A distributed key-value store is the standard
solution. Redis is the most common choice, but Valkey is an open-source, Redis-compatible
fork that avoids licensing concerns and has an official Go client with no CGo dependency.

## Decision
Use Valkey as the session store via `github.com/valkey-io/valkey-go`. Session IDs are stored
in HTTP-only cookies; session payloads (UserID, role, expiry) are serialised to JSON and
stored in Valkey under the session ID key with a TTL. The `pkg/platform/session` package
wraps the client and exposes typed helpers (`session.UserID(c)`, `session.Set(c, ...)`, etc.).

## Consequences
**Easier:**
- Sessions survive server restarts and work across multiple server instances.
- TTL-based expiry is handled natively by Valkey — no background cleanup job needed.
- valkey-go client is pure Go; no CGo, no additional system dependencies beyond the server.

**Harder:**
- Valkey is an additional infrastructure dependency (managed via docker-compose in dev).
- Network latency for every authenticated request (one Valkey GET per request).

**Deferred:**
- Session clustering / Valkey sentinel for HA — docker-compose uses a single node.
- Sliding window TTL renewal on each request — current implementation uses fixed TTL.
