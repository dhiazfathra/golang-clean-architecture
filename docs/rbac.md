# RBAC — Role-Based Access Control

## Permission Model

A permission is a triple:

```go
type Permission struct {
    Module string      // e.g. "user", "order", "*" (wildcard)
    Action string      // e.g. "create", "read", "list", "delete", "*" (wildcard)
    Fields FieldPolicy
}

type FieldPolicy struct {
    Mode   string   // "all" | "allow" | "deny"
    Fields []string // field names (empty when mode is "all")
}
```

Permissions are stored as events on role aggregates and projected into `rbac_roles_read` and `rbac_permissions_read`.

---

## Field Policies

| Mode | Meaning |
|------|---------|
| `all` | Caller sees every field in the response |
| `allow` | Caller sees **only** the listed fields |
| `deny` | Caller sees every field **except** the listed fields |

**Union semantics when a caller has multiple roles:**
- All matching permissions (same module + action) are collected.
- `all` anywhere → caller sees every field (most permissive wins).
- Otherwise merge: `allow` lists are unioned; `deny` lists are unioned.
- `rbac.FilterResponse` applies the merged policy before serialising JSON.

---

## Role Assignment

```
POST /admin/roles/:id/permissions   ← GrantPermission event on role aggregate
                                       → rbac_permissions_read UPSERT

GET /admin/users/:id/roles          ← reads user_roles_read
POST /users (create user)           ← service optionally assigns default role
                                       via RoleAssigned event on user aggregate
                                       → user_roles_read UPSERT
```

`RoleAssigned` is an event on the **user** aggregate (not the role aggregate).
The user's role membership is projected into `user_roles_read`.

---

## Middleware Flow

```
HTTP request
  │
  ├─ session.RequireSession
  │    reads cookie → Valkey → stores userID in echo.Context
  │
  ├─ rbac.RequirePermission(svc, module, action)
  │    1. session.UserID(c) → userID
  │    2. rbacRepo.ListUserRoles(userID) → []roleID
  │    3. rbacRepo.ListPermissionsForRoles(roleIDs, module, action)
  │    4. any match? → next()   else → 403
  │
  ▼
Handler executes
  │
  └─ rbac.FilterResponse(c, result)
       1. same userID + role lookup
       2. merge FieldPolicy from all matching permissions
       3. marshal result → map → strip/keep fields → re-marshal → c.JSON(200, …)
```

---

## Seeder Conventions

The seeder (`pkg/platform/seeder`) runs at startup and is idempotent (events are only appended if the role/user does not yet exist):

| Seeded entity | Description |
|---------------|-------------|
| `super_admin` role | Wildcard permission `{module: "*", action: "*", fields: {mode: "all"}}` |
| `admin@system.local` user | Assigned `super_admin`; password from `SEED_SUPER_ADMIN_PASSWORD` |
| `<module>_admin` role (per registered module) | Full CRUD permissions on that module; all fields |
| `<module>@system.local` user | Assigned `<module>_admin`; password from `SEED_DEFAULT_MODULE_PASSWORD` |

---

## Admin Endpoints

All endpoints under `/admin/…` require an active session (`RequireSession` middleware on the `/admin` group).

### Role management (requires `rbac:manage`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/admin/roles` | Create a new role |
| GET | `/admin/roles` | List all roles |
| GET | `/admin/roles/:id` | Get a single role with its permissions |
| DELETE | `/admin/roles/:id` | Soft-delete a role |
| POST | `/admin/roles/:id/permissions` | Grant a permission to a role |
| DELETE | `/admin/roles/:id/permissions/:perm` | Revoke a permission from a role |
| GET | `/admin/users/:id/roles` | List roles assigned to a user |

### Audit (requires `audit:read`)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/audit/:type/:id` | Full event history for any aggregate |

The `:type` segment is the aggregate type (e.g. `user`, `order`, `role`).
The `:id` segment is the Snowflake aggregate ID.
