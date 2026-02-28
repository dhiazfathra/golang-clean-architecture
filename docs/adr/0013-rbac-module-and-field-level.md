# ADR-0013: RBAC with module-level and field-level permissions

## Status
Accepted

## Context
Access control requirements go beyond simple role checks. Different roles need access to
different subsets of fields within the same resource (e.g., a manager can see salary data,
a peer cannot). A flat permission model (`can_read_user: true/false`) cannot express this.
Additionally, since RBAC configuration itself is sensitive and auditable, it should be
managed through the same event-sourced mechanism as other domain state.

## Decision
RBAC is implemented as an event-sourced module in `pkg/platform/rbac`. Each `Permission`
struct has three parts: `Module` (e.g., `"user"`), `Action` (e.g., `"read"`), and a
`FieldPolicy` that specifies `Mode` (`"all"`, `"allow"`, `"deny"`) and an optional field
list. Roles aggregate permissions and are stored as event-sourced aggregates. Route
protection uses the `rbac.RequirePermission(svc, module, action)` middleware. Field-level
filtering uses `rbac.FilterResponse(c, result)` in handlers (see ADR-0014).

## Consequences
**Easier:**
- Fine-grained field-level access control without per-handler code.
- RBAC changes (new role, permission update) are fully auditable via the event store.
- `FieldPolicy` is declarative — adding a new field to a resource does not require RBAC
  code changes; update the policy data.

**Harder:**
- More complex than a simple `can_do_X` boolean check; developers must understand
  `FieldPolicy.Mode` semantics.
- Event-sourced RBAC means a role's effective permissions require replaying its events
  on startup (mitigated by snapshot and in-memory cache after load).

**Deferred:**
- Attribute-based access control (ABAC) — the current model is pure RBAC. If row-level
  filtering (e.g., "users can only see their own orders") is needed, an ABAC extension
  would be layered on top.
