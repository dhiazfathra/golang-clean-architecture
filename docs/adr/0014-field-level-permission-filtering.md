# ADR-0014: Field-level permission filtering via generic JSON response filter

## Status
Accepted

## Context
Once field-level permissions are defined (ADR-0013), they must be enforced at the API
response boundary. Options include: per-handler manual field omission (error-prone and
verbose), struct-tag-based view models per role (combinatorial explosion), or a generic
filter that operates on any response shape. The generic approach is the most maintainable
because it is driven entirely by the `FieldPolicy` data, not handler code.

## Decision
`rbac.FilterResponse(c echo.Context, v any) any` is called as the last step in every GET
handler before `c.JSON(...)`. It:
1. Marshals `v` to a JSON intermediate (`map[string]any` or `[]map[string]any`).
2. Looks up the caller's role and the `FieldPolicy` for the current module+action.
3. Applies allow-list or deny-list filtering on the map keys.
4. Returns the filtered value for serialisation.

`PageResponse[T]` is handled transparently: the filter recurses into the `.Items` slice.
If the role has `Mode: "all"`, the original value is returned unchanged (no serialisation
overhead).

## Consequences
**Easier:**
- One-liner in every GET handler: `return c.JSON(200, rbac.FilterResponse(c, result))`.
- No per-handler knowledge of which fields to include/exclude.
- New fields added to a struct are automatically included in `"all"` mode and can be
  added to allow/deny lists in policy data without code changes.

**Harder:**
- Double JSON serialisation (marshal to intermediate map, then marshal map to response)
  adds latency. Acceptable for typical payload sizes; may need optimisation for
  high-throughput bulk endpoints.
- JSON-tag-driven: struct fields without `json` tags are not filterable.

**Deferred:**
- Zero-copy field filtering using `encoding/json` streaming for large response payloads.
- `omitempty` semantics interaction with the filter (currently filtered fields are absent,
  consistent with `omitempty` behaviour).
