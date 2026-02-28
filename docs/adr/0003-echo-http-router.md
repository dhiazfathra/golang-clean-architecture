# ADR-0003: Echo as HTTP router

## Status
Accepted

## Context
The project needs an HTTP framework with: middleware chaining, typed context passing, route
grouping, and a healthy ecosystem. The standard library `net/http` is sufficient for basic
routing but requires significant boilerplate for middleware and parameter extraction. A
thin, well-maintained framework avoids this boilerplate without introducing hidden magic.

## Decision
Use Echo v4 (`github.com/labstack/echo/v4`) as the HTTP router and middleware host.
Echo's `echo.Context` is passed through every handler. Middleware is registered at router
level so modules receive clean handlers with no framework imports beyond Echo.

## Consequences
**Easier:**
- Rich middleware ecosystem (CORS, recover, request-id, etc.) available out of the box.
- Named route parameters and query binding via `c.Param` / `c.Bind`.
- Route grouping (`e.Group("/api/v1")`) simplifies module registration.
- Observability middleware wraps Echo natively (dd-trace-go Echo integration, ADR-0011).

**Harder:**
- Handlers are coupled to `echo.Context` rather than standard `http.ResponseWriter`/`*http.Request`.
  Switching frameworks would require handler rewrites.

**Deferred:**
- HTTP/2 server push and WebSocket support — can be added via Echo middleware if needed.
