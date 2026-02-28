# ADR-0018: Scalar as the API documentation renderer

## Status
Accepted

## Context
As the service grows to cover multiple modules (auth, user, order, and any module produced by
the CLI generator), developers integrating with the API need a single, interactive reference
they can consult without reading source code.

Three rendering options were evaluated:

1. **Swagger UI** (`swaggo/swag` + `swaggo/echo-swagger`) — generates OpenAPI from struct
   annotations; the de-facto Go standard. However, the UI is dated, annotation syntax is
   verbose, and the code-gen step tightly couples the API shape to internal type names.
   Annotations must be kept in sync with handler signatures; drift is not caught at compile time.

2. **Redoc** — polished, read-only reference with strong left-panel navigation. No additional
   Go dependency: served via a single HTML page. No interactive playground, so developers
   must reach for a separate tool (curl, Postman) to test endpoints.

3. **Scalar** — modern, open-source API reference UI with a built-in HTTP playground that
   supports OpenAPI 3.x. Can be self-hosted by serving one HTML page that loads the Scalar
   JavaScript from a CDN. No new Go library is required. The UI is actively maintained and
   widely adopted as a Swagger UI replacement.

The project follows a "minimal magic, minimal dependencies" philosophy (ADR-0007). A
manually-maintained `api/openapi.yaml` kept as the canonical source of truth aligns with this
philosophy better than annotation-driven code generation. The spec can be linted, diffed, and
reviewed independently of Go code.

A `GET /health` liveness probe and `GET /health/ready` readiness probe are added in the same
milestone: they are the first endpoints a new consumer (or a load balancer) needs, and they
belong in the API reference alongside the business endpoints.

## Decision

1. **Renderer** — Use Scalar for API documentation.

2. **Spec format** — Maintain a hand-authored `api/openapi.yaml` (OpenAPI 3.x) as the
   canonical specification. It is embedded into the binary via `//go:embed` in `api/embed.go`.

3. **Docs routes** — Register `GET /docs` (Scalar HTML) and `GET /openapi.yaml` (raw spec)
   on the public Echo group. Both routes are gated behind `config.Env != "production"` so that
   internal API shapes are not exposed by default in production deployments.

4. **Health routes** — Register `GET /health` (liveness) and `GET /health/ready` (readiness)
   on the root Echo instance — outside all auth middleware — so that load balancers and
   orchestrators can always reach them. The readiness probe pings both PostgreSQL and Valkey
   with a 3-second timeout and returns `503` if either is unreachable.

5. **CLI generator extension** — The M7 CLI generator is extended with a new template
   `templates/module/openapi_paths.yaml.tmpl`. Running `make generate` now also emits
   `api/paths/{module}.yaml`, a ready-to-merge OpenAPI path-and-schema fragment. Developers
   paste the fragment into `api/openapi.yaml` after verifying the generated handlers.

6. **No compile-time contract** — The spec is not automatically verified against handler
   signatures. This is an accepted trade-off; the spec is maintained as living documentation.
   A future milestone may add contract testing (e.g. `oapi-codegen` validation middleware) if
   drift becomes a problem.

## Consequences

**Easier:**
- Developers get a beautiful, interactive API reference with a built-in playground at `/docs`
  with no new Go runtime dependency.
- The OpenAPI spec is a first-class artefact: it can be linted with `vacuum` or `spectral`,
  shared with API consumers, and kept under version control like any other source file.
- The health endpoints satisfy load balancer requirements without any framework changes.
- New generated modules get an OpenAPI path fragment automatically from `make generate`;
  integrating it into the root spec is a single copy-paste.

**Harder:**
- The spec must be kept in sync with handler signatures manually (or via the path fragment
  approach). There is no compile-time guarantee that the spec matches the actual routes.
- Developers must update `api/openapi.yaml` whenever a handler signature changes; this is
  a process concern, not a technical one.

**Deferred:**
- If auto-generation becomes important, `swaggo` annotations or `oapi-codegen` can be added
  later without changing the Scalar renderer — it consumes whatever spec is served at
  `/openapi.yaml`.
- Contract-testing middleware (validate requests/responses against the spec at runtime) can
  be added in a future hardening milestone.
- Custom Scalar theme configuration (logo, colours, `x-scalar-` extensions) can be applied
  by editing `api/scalar.html` with no Go code changes.
