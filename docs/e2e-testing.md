# End-to-End Testing Guide

This guide demonstrates how to test all API endpoints using `curl`. All examples assume the server is running on `http://localhost:8080`.

---

## Prerequisites

```bash
# Start the server with seeded data
make setup

# Or manually:
make infra-up
make migrate
make seed
make run
```

**Default seeded credentials:**

- Super Admin: `admin@system.local` / `secret123` (if `SEED_SUPER_ADMIN_PASSWORD=secret123`)
- Module Admins: `user-admin@system.local`, `order-admin@system.local` / `module123` (if `SEED_DEFAULT_MODULE_PASSWORD=module123`)

---

## 1. Health Checks

### Liveness Probe

```bash
curl -i http://localhost:8080/health
```

**Expected:** `200 OK` with `{"status":"ok"}`

### Readiness Probe

```bash
curl -i http://localhost:8080/health/ready
```

**Expected:** `200 OK` if Postgres + Valkey are reachable, `503 Service Unavailable` otherwise.

---

## 2. Authentication

### Login (Session Cookie)

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@system.local",
    "password": "secret123"
  }' \
  -c cookies.txt
```

**Expected:** `200 OK` with `Set-Cookie: session_id=...`

**Save the cookie** to `cookies.txt` for subsequent requests.

### Get Current User

```bash
curl -i http://localhost:8080/api/v1/auth/me \
  -b cookies.txt
```

**Expected:** `200 OK` with user object including `id`, `email`, `roles`.

### Logout

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/logout \
  -b cookies.txt
```

**Expected:** `200 OK`, session invalidated.

---

## 3. Users Module

### Create User

```bash
curl -i -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "email": "testuser@example.com",
    "password": "password123",
    "full_name": "Test User"
  }'
```

**Expected:** `201 Created` with user object including `id`.

**Permission required:** `user:create`

**Save the returned `id`** for subsequent requests (e.g., `USER_ID=123456789`).

### List Users

```bash
curl -i http://localhost:8080/api/v1/users \
  -b cookies.txt
```

**Expected:** `200 OK` with array of user objects. Fields filtered by caller's RBAC permissions.

**Permission required:** `user:list`

### Get User by ID

```bash
curl -i http://localhost:8080/api/v1/users/123456789 \
  -b cookies.txt
```

**Expected:** `200 OK` with user object. Fields filtered by caller's RBAC permissions.

**Permission required:** `user:read`

### Get User by ID (Admin, includes soft-deleted)

```bash
curl -i http://localhost:8080/api/v1/admin/users/123456789 \
  -b cookies.txt
```

**Expected:** `200 OK` with user object, even if soft-deleted.

**Permission required:** `user:read`

### Delete User (Soft Delete)

```bash
curl -i -X DELETE http://localhost:8080/api/v1/users/123456789 \
  -b cookies.txt
```

**Expected:** `204 No Content`

**Permission required:** `user:delete`

---

## 4. Orders Module

### Create Order

```bash
curl -i -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "customer_name": "John Doe",
    "total_amount": 99.99
  }'
```

**Expected:** `201 Created` with order object including `id`.

**Permission required:** `order:create`

**Save the returned `id`** for subsequent requests (e.g., `ORDER_ID=987654321`).

### List Orders

```bash
curl -i http://localhost:8080/api/v1/orders \
  -b cookies.txt
```

**Expected:** `200 OK` with array of order objects. Fields filtered by caller's RBAC permissions.

**Permission required:** `order:list`

### Get Order by ID

```bash
curl -i http://localhost:8080/api/v1/orders/987654321 \
  -b cookies.txt
```

**Expected:** `200 OK` with order object. Fields filtered by caller's RBAC permissions.

**Permission required:** `order:read`

### Delete Order (Soft Delete)

```bash
curl -i -X DELETE http://localhost:8080/api/v1/orders/987654321 \
  -b cookies.txt
```

**Expected:** `204 No Content`

**Permission required:** `order:delete`

---

## 5. RBAC (Admin)

### Create Role

```bash
curl -i -X POST http://localhost:8080/api/v1/admin/roles \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "name": "viewer",
    "description": "Read-only access"
  }'
```

**Expected:** `201 Created` with role object including `id`.

**Permission required:** `rbac:manage`

**Save the returned `id`** for subsequent requests (e.g., `ROLE_ID=111222333`).

### List Roles

```bash
curl -i http://localhost:8080/api/v1/admin/roles \
  -b cookies.txt
```

**Expected:** `200 OK` with array of role objects.

**Permission required:** `rbac:manage`

### Get Role by ID

```bash
curl -i http://localhost:8080/api/v1/admin/roles/111222333 \
  -b cookies.txt
```

**Expected:** `200 OK` with role object including permissions.

**Permission required:** `rbac:manage`

### Grant Permission to Role

```bash
curl -i -X POST http://localhost:8080/api/v1/admin/roles/111222333/permissions \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "module": "user",
    "action": "read",
    "fields": ["id", "email", "full_name"]
  }'
```

**Expected:** `201 Created`

**Permission required:** `rbac:manage`

**Notes:**

- `fields` is optional; omit for full access to all fields
- Common actions: `create`, `read`, `update`, `delete`, `list`

### Revoke Permission from Role

```bash
curl -i -X DELETE "http://localhost:8080/api/v1/admin/roles/111222333/permissions/user:read" \
  -b cookies.txt
```

**Expected:** `204 No Content`

**Permission required:** `rbac:manage`

**Format:** `{module}:{action}` (URL-encoded if necessary)

### Get User Roles

```bash
curl -i http://localhost:8080/api/v1/admin/users/123456789/roles \
  -b cookies.txt
```

**Expected:** `200 OK` with array of role objects assigned to the user.

**Permission required:** `rbac:manage`

### Delete Role

```bash
curl -i -X DELETE http://localhost:8080/api/v1/admin/roles/111222333 \
  -b cookies.txt
```

**Expected:** `204 No Content`

**Permission required:** `rbac:manage`

---

## 6. API Tokens (Admin, Session-Only)

API tokens enable programmatic access (CI/CD, SDKs, CLI tools). Tokens are prefixed with `gca_` and can be used as `Bearer` auth on multi-auth endpoints.

### Create API Token

```bash
curl -i -X POST http://localhost:8080/api/v1/admin/api-tokens \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "name": "CI/CD Token",
    "description": "Token for automated deployments"
  }'
```

**Expected:** `201 Created` with token object including `token` (raw value, shown only once).

**Permission required:** `apitoken:manage`

**Save the `token` value** (e.g., `TOKEN=gca_abc123...`). You cannot retrieve it again.

### List API Tokens

```bash
curl -i http://localhost:8080/api/v1/admin/api-tokens \
  -b cookies.txt
```

**Expected:** `200 OK` with array of token objects (raw token values are **not** included).

**Permission required:** `apitoken:manage`

### Delete API Token

```bash
curl -i -X DELETE http://localhost:8080/api/v1/admin/api-tokens/TOKEN_ID \
  -b cookies.txt
```

**Expected:** `204 No Content`

**Permission required:** `apitoken:manage`

---

## 7. Feature Flags (Multi-Auth)

Feature flags use a 3-tier cache: **sync.Map → Valkey → Postgres**.

### Create Feature Flag

```bash
# Using session cookie
curl -i -X POST http://localhost:8080/api/v1/admin/feature-flags \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "key": "new_checkout_flow",
    "enabled": true,
    "description": "Enable new checkout UI"
  }'

# Using Bearer token
curl -i -X POST http://localhost:8080/api/v1/admin/feature-flags \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer gca_abc123..." \
  -d '{
    "key": "new_checkout_flow",
    "enabled": true,
    "description": "Enable new checkout UI"
  }'
```

**Expected:** `201 Created` with feature flag object.

**Permission required:** `featureflag:manage`

### List Feature Flags

```bash
curl -i http://localhost:8080/api/v1/admin/feature-flags \
  -b cookies.txt
```

**Expected:** `200 OK` with array of feature flag objects.

**Permission required:** `featureflag:manage`

### Update Feature Flag

```bash
curl -i -X PATCH http://localhost:8080/api/v1/admin/feature-flags/new_checkout_flow \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "enabled": false
  }'
```

**Expected:** `200 OK` with updated feature flag object.

**Permission required:** `featureflag:manage`

### Delete Feature Flag

```bash
curl -i -X DELETE http://localhost:8080/api/v1/admin/feature-flags/new_checkout_flow \
  -b cookies.txt
```

**Expected:** `204 No Content`

**Permission required:** `featureflag:manage`

---

## 8. Environment Variables (Multi-Auth)

Dynamic environment variables scoped by platform (`mobile`, `web`, `be`, etc.). Uses the same 3-tier cache as feature flags.

### Create Environment Variable

```bash
# Using session cookie
curl -i -X POST http://localhost:8080/api/v1/envs \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "platform": "mobile",
    "key": "API_BASE_URL",
    "value": "https://api.example.com",
    "description": "Base URL for API calls"
  }'

# Using Bearer token
curl -i -X POST http://localhost:8080/api/v1/envs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer gca_abc123..." \
  -d '{
    "platform": "mobile",
    "key": "API_BASE_URL",
    "value": "https://api.example.com",
    "description": "Base URL for API calls"
  }'
```

**Expected:** `201 Created` with environment variable object.

**Permission required:** `envvar:manage`

### Get Environment Variable by Platform and Key

```bash
curl -i http://localhost:8080/api/v1/envs/mobile/API_BASE_URL \
  -b cookies.txt
```

**Expected:** `200 OK` with environment variable object.

**Permission required:** `envvar:manage`

### List Environment Variables by Platform

```bash
curl -i http://localhost:8080/api/v1/envs/mobile \
  -b cookies.txt
```

**Expected:** `200 OK` with array of environment variable objects for the specified platform.

**Permission required:** `envvar:manage`

### Update Environment Variable

```bash
curl -i -X PUT http://localhost:8080/api/v1/envs/mobile/API_BASE_URL \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "value": "https://api-v2.example.com",
    "description": "Updated base URL"
  }'
```

**Expected:** `200 OK` with updated environment variable object.

**Permission required:** `envvar:manage`

### Delete Environment Variable

```bash
curl -i -X DELETE http://localhost:8080/api/v1/envs/mobile/API_BASE_URL \
  -b cookies.txt
```

**Expected:** `204 No Content`

**Permission required:** `envvar:manage`

---

## 9. Audit

### Get Audit Trail for Aggregate

```bash
curl -i http://localhost:8080/api/v1/admin/audit/user/123456789 \
  -b cookies.txt
```

**Expected:** `200 OK` with array of event objects showing the full event-sourced history.

**Permission required:** `audit:read`

**Parameters:**

- `:type` — aggregate type (e.g., `user`, `order`, `role`)
- `:id` — aggregate ID (Snowflake int64)

---

## 10. API Documentation (Non-Production Only)

### Scalar API UI

```bash
open http://localhost:8080/docs
```

**Expected:** Interactive API documentation UI (Scalar).

**Note:** Only available when `ENV != production`.

### Raw OpenAPI Spec

```bash
curl -i http://localhost:8080/openapi.yaml
```

**Expected:** `200 OK` with OpenAPI 3.1 YAML spec.

**Note:** Only available when `ENV != production`.

---

## Complete E2E Test Flow

This example demonstrates a full workflow: login → create user → assign role → create order → audit trail → logout.

```bash
# 1. Login as super admin
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@system.local","password":"secret123"}' \
  -c cookies.txt

# 2. Create a new role
ROLE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/admin/roles \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"order_viewer","description":"Can view orders"}')
ROLE_ID=$(echo $ROLE_RESPONSE | jq -r '.id')

# 3. Grant order:read permission to the role
curl -X POST http://localhost:8080/api/v1/admin/roles/$ROLE_ID/permissions \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"module":"order","action":"read","fields":["id","customer_name","total_amount"]}'

# 4. Create a new user
USER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"email":"viewer@example.com","password":"pass123","full_name":"Order Viewer"}')
USER_ID=$(echo $USER_RESPONSE | jq -r '.id')

# 5. Create an order
ORDER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"customer_name":"Jane Smith","total_amount":149.99}')
ORDER_ID=$(echo $ORDER_RESPONSE | jq -r '.id')

# 6. Get the order (should see all fields as super admin)
curl -s http://localhost:8080/api/v1/orders/$ORDER_ID \
  -b cookies.txt | jq

# 7. View audit trail for the order
curl -s http://localhost:8080/api/v1/admin/audit/order/$ORDER_ID \
  -b cookies.txt | jq

# 8. Create an API token
TOKEN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/admin/api-tokens \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"Test Token","description":"For testing"}')
TOKEN=$(echo $TOKEN_RESPONSE | jq -r '.token')

# 9. Use the token to create a feature flag
curl -X POST http://localhost:8080/api/v1/admin/feature-flags \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"key":"test_flag","enabled":true,"description":"Test flag"}'

# 10. Logout
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -b cookies.txt
```

---

## Testing RBAC Field-Level Permissions

```bash
# 1. Login as user with limited permissions
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"viewer@example.com","password":"pass123"}' \
  -c viewer_cookies.txt

# 2. Get an order (should only see permitted fields: id, customer_name, total_amount)
curl -s http://localhost:8080/api/v1/orders/$ORDER_ID \
  -b viewer_cookies.txt | jq

# 3. Try to create an order (should fail with 403 Forbidden)
curl -i -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -b viewer_cookies.txt \
  -d '{"customer_name":"Test","total_amount":50.00}'
```

---

## Testing Multi-Auth (Session vs Bearer Token)

```bash
# Create a token (requires session auth)
TOKEN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/admin/api-tokens \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"Multi-Auth Test","description":"Testing both auth methods"}')
TOKEN=$(echo $TOKEN_RESPONSE | jq -r '.token')

# Test feature flag endpoint with session cookie
curl -i http://localhost:8080/api/v1/admin/feature-flags \
  -b cookies.txt

# Test feature flag endpoint with Bearer token
curl -i http://localhost:8080/api/v1/admin/feature-flags \
  -H "Authorization: Bearer $TOKEN"

# Test env var endpoint with Bearer token
curl -i http://localhost:8080/api/v1/envs/mobile \
  -H "Authorization: Bearer $TOKEN"
```

---

## Error Cases to Test

### 401 Unauthorized (No Auth)

```bash
curl -i http://localhost:8080/api/v1/users
# Expected: 401 Unauthorized
```

### 403 Forbidden (Insufficient Permissions)

```bash
# Login as a user without rbac:manage permission
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"viewer@example.com","password":"pass123"}' \
  -c limited_cookies.txt

# Try to create a role
curl -i -X POST http://localhost:8080/api/v1/admin/roles \
  -H "Content-Type: application/json" \
  -b limited_cookies.txt \
  -d '{"name":"test","description":"Should fail"}'
# Expected: 403 Forbidden
```

### 404 Not Found

```bash
curl -i http://localhost:8080/api/v1/users/999999999999 \
  -b cookies.txt
# Expected: 404 Not Found
```

### 400 Bad Request (Invalid Input)

```bash
curl -i -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"email":"invalid-email","password":"short"}'
# Expected: 400 Bad Request with validation errors
```

---

## Tips

1. **Use `jq` for JSON parsing**: Install with `brew install jq` (macOS) or `apt install jq` (Linux)
2. **Save cookies**: Use `-c cookies.txt` to save and `-b cookies.txt` to send session cookies
3. **Pretty-print JSON**: Pipe responses through `jq` for readable output
4. **Check response headers**: Use `-i` flag to see HTTP status codes and headers
5. **Extract IDs**: Use `jq -r '.id'` to extract IDs from JSON responses for subsequent requests
6. **Test async projections**: After creating events, wait ~500ms for projections to complete before querying read models
7. **Monitor logs**: Run `docker-compose logs -f app` to see server logs during testing
8. **Reset state**: Use `make setup-reset` to tear down and rebuild the entire environment

---

## Automated Testing with Scripts

Create a test script `test-e2e.sh`:

```bash
#!/bin/bash
set -e

BASE_URL="http://localhost:8080"
COOKIES="test_cookies.txt"

echo "==> Testing health endpoints"
curl -sf $BASE_URL/health > /dev/null
curl -sf $BASE_URL/health/ready > /dev/null
echo "✓ Health checks passed"

echo "==> Testing authentication"
curl -sf -X POST $BASE_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@system.local","password":"secret123"}' \
  -c $COOKIES > /dev/null
echo "✓ Login successful"

echo "==> Testing user creation"
USER_ID=$(curl -sf -X POST $BASE_URL/api/v1/users \
  -H "Content-Type: application/json" \
  -b $COOKIES \
  -d '{"email":"test@example.com","password":"pass123","full_name":"Test User"}' \
  | jq -r '.id')
echo "✓ User created: $USER_ID"

echo "==> Testing order creation"
ORDER_ID=$(curl -sf -X POST $BASE_URL/api/v1/orders \
  -H "Content-Type: application/json" \
  -b $COOKIES \
  -d '{"customer_name":"Test Customer","total_amount":99.99}' \
  | jq -r '.id')
echo "✓ Order created: $ORDER_ID"

echo "==> Testing audit trail"
curl -sf $BASE_URL/api/v1/admin/audit/order/$ORDER_ID \
  -b $COOKIES > /dev/null
echo "✓ Audit trail retrieved"

echo "==> Cleaning up"
curl -sf -X DELETE $BASE_URL/api/v1/users/$USER_ID -b $COOKIES > /dev/null
curl -sf -X DELETE $BASE_URL/api/v1/orders/$ORDER_ID -b $COOKIES > /dev/null
rm -f $COOKIES
echo "✓ Cleanup complete"

echo ""
echo "All E2E tests passed! ✓"
```

Run with:

```bash
chmod +x test-e2e.sh
./test-e2e.sh
```

---

## Next Steps

- **Integration tests**: Write Go tests using `net/http/httptest` for programmatic endpoint testing
- **Load testing**: Use `hey`, `wrk`, or `k6` to test performance under load
- **Contract testing**: Use Pact or similar tools for consumer-driven contract tests
- **Monitoring**: Set up Datadog dashboards to track API metrics, error rates, and latency
