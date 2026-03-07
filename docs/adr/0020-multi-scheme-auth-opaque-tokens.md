# ADR-0020: Multi-scheme authentication with opaque Bearer tokens

## Status
Accepted

## Context
The platform's APIs (feature flags, environment variables) are protected by session-based
authentication, which requires a browser cookie. This works well for human operators using a
web UI, but creates friction for programmatic access patterns:

1. **CI/CD pipelines** need to toggle feature flags or update environment variables as part of
   deployment workflows. Maintaining a session cookie across pipeline steps is fragile and
   requires storing session credentials in CI secrets.

2. **SDKs and CLI tools** that integrate with the platform need a stateless authentication
   mechanism. Session cookies are browser-centric and awkward to manage in non-browser clients.

3. **Machine-to-machine communication** (e.g., microservices querying feature flags) benefits
   from long-lived, revocable credentials rather than session tokens with short TTLs.

Two approaches were considered:

1. **JWT tokens** — self-contained, stateless, verifiable without a database lookup. However,
   JWTs cannot be revoked before expiry without maintaining a blocklist (negating the stateless
   benefit). They also risk token size bloat when embedding claims and require careful key
   management (rotation, asymmetric vs. symmetric).

2. **Opaque tokens with server-side lookup** — a random string mapped to a user via a hashed
   lookup in the existing 3-tier cache (`sync.Map` -> Valkey -> Postgres). Revocation is
   instant (delete from cache + DB). The 3-tier cache infrastructure already exists for feature
   flags and environment variables (ADR-0019), so no new caching pattern is introduced.

The platform already has Valkey deployed (ADR-0005) and the `kvstore.Store` abstraction
(from M21) provides a reusable 3-tier cache layer, making opaque tokens the simpler choice
with better revocation semantics.

## Decision

1. **Opaque token format** — Tokens are 68-character strings: a `gca_` prefix (for easy
   identification and grep-ability) followed by 64 hex characters from 32 bytes of
   `crypto/rand`. Only the SHA-256 hash is stored; the raw token is returned to the user
   exactly once at creation time.

2. **3-tier cached validation** — Token validation uses `kvstore.Store` with the same
   `sync.Map -> Valkey -> Postgres` pattern as feature flags and environment variables.
   The store maps `SHA-256(rawToken)` to `userID`. Background refresh keeps all instances
   consistent within the configured interval.

3. **Multi-scheme middleware** — A new `RequireMultiAuth` middleware in the `session` package
   accepts either a `Bearer` token in the `Authorization` header or a session cookie:
   - Bearer token is checked first (stateless clients don't send cookies).
   - On success, `user_id` and `auth_method` ("token" or "session") are set in the Echo
     context. `session.UserID(c)` works identically for both auth methods.
   - The `TokenValidator` interface is defined in the `session` package to avoid import
     cycles between `session` and `apitoken`.

4. **Selective route migration** — Only `featureflag` and `envvar` routes are moved to the
   `RequireMultiAuth` middleware. Token management endpoints (`/admin/api-tokens`) remain
   session-only — tokens cannot create other tokens.

5. **Token lifecycle** — Tokens have a mandatory expiry (`expires_at`), support soft delete
   for revocation (following ADR-0012), and are scoped to the creating user. RBAC permissions
   are evaluated against the token's associated `userID`, so token-authenticated requests
   have the same permission boundaries as session-authenticated ones.

6. **Package location** — `pkg/platform/apitoken/` follows the platform package convention,
   with the standard `model.go`, `repository.go`, `service.go`, `handler.go`, `routes.go`
   layout.

## Consequences

**Easier:**
- CI/CD pipelines and CLI tools can authenticate with a simple `Authorization: Bearer gca_...`
  header, with no session management required.
- Token revocation is instant — deleting from the cache and database takes effect immediately
  on the local instance and within one refresh interval on other instances.
- The `gca_` prefix makes tokens easy to identify in logs, secret scanners, and credential
  rotation tools.
- No new infrastructure: the same Valkey instance and `kvstore.Store` pattern are reused.
- RBAC enforcement is unchanged — token-authenticated requests go through the same permission
  checks as session-authenticated ones.

**Harder:**
- Operators must understand that raw tokens are shown only once at creation. Lost tokens
  cannot be recovered and must be revoked and recreated.
- Token validation adds a hash computation per request on the Bearer path (SHA-256 of the
  raw token), though this is negligible compared to network I/O.
- The `RequireMultiAuth` middleware is slightly more complex than `RequireSession`, as it
  must handle two authentication paths and set the `auth_method` context value.

**Deferred:**
- Token scoping (restricting a token to specific modules or actions beyond RBAC) can be added
  by extending the `api_tokens` table with a `scopes` column.
- Rate limiting per token can be implemented using the token hash as the rate-limit key.
- Token rotation (issuing a new token that supersedes an old one with a grace period) can be
  built on top of the existing create/revoke operations.
- An audit trail of token usage (which token accessed which endpoint) can be added by logging
  the token prefix in request middleware.
