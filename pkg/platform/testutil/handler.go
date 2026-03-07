package testutil

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo/v4"
)

// CtxOption is a functional option that mutates an Echo context after construction.
type CtxOption func(echo.Context)

// WithUserID injects the given user ID into the Echo context under the "user_id" key.
func WithUserID(id string) CtxOption {
	return func(c echo.Context) {
		c.Set("user_id", id)
	}
}

// WithFieldPolicy injects any value as the RBAC field policy.
// The caller supplies the value, so testutil stays import-free.
func WithFieldPolicy(policy any) CtxOption {
	return func(c echo.Context) {
		c.Set("rbac_field_policy", policy)
	}
}

// EchoCtx creates a new Echo context and response recorder for use in HTTP handler tests.
// It constructs an HTTP request with the given method and target URL. If body is non-empty,
// the request body is set to the provided JSON string and the Content-Type header is set to
// application/json. If body is empty, the request is created with no body.
//
// Optional CtxOption functions can be passed to mutate the context after construction,
// e.g. to inject authentication or authorization state.
//
// Parameters:
//   - method: HTTP method (e.g., http.MethodGet, http.MethodPost)
//   - target: request URL or path (e.g., "/api/users")
//   - body:   JSON request body as a string; pass "" for requests with no body
//   - opts:   zero or more CtxOption functions applied to the context in order
//
// Returns the Echo context and a response recorder that captures the handler's response.
func EchoCtx(method, target, body string, opts ...CtxOption) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	for _, opt := range opts {
		opt(c)
	}
	return c, rec
}

// AuthedEchoCtx creates an authenticated Echo context for use in HTTP handler tests.
// It is a convenience wrapper around EchoCtx that injects a "user_id" of "actor_1".
//
// Parameters:
//   - method: HTTP method (e.g., http.MethodGet, http.MethodPost)
//   - target: request URL or path (e.g., "/api/users")
//   - body:   JSON request body as a string; pass "" for requests with no body
//
// Returns the Echo context (with user_id set to "actor_1") and a response recorder.
func AuthedEchoCtx(method, target, body string) (echo.Context, *httptest.ResponseRecorder) {
	return EchoCtx(method, target, body, WithUserID("actor_1"))
}

// AuthedEchoCtxWithPolicy creates an authenticated Echo context with a field-level RBAC policy
// attached, for use in HTTP handler tests that require both authentication and authorization.
// It is a convenience wrapper around EchoCtx that injects a "user_id" of "actor_1" and grants
// access to all fields via rbac.AllFields().
//
// Parameters:
//   - method: HTTP method (e.g., http.MethodGet, http.MethodPost)
//   - target: request URL or path (e.g., "/api/users")
//   - body:   JSON request body as a string; pass "" for requests with no body
//
// Returns the Echo context (with user_id and rbac_field_policy set) and a response recorder.
func AuthedEchoCtxWithPolicy(method, target, body string, policy any) (echo.Context, *httptest.ResponseRecorder) {
	return EchoCtx(method, target, body, WithUserID("actor_1"), WithFieldPolicy(policy))
}
