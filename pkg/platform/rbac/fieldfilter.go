package rbac

import (
	"encoding/json"

	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
)

func FilterResponse(c echo.Context, v any) any {
	policy, ok := c.Get("rbac_field_policy").(FieldPolicy)
	if !ok || policy.Mode == "all" {
		return v
	}
	return filterByPolicy(v, policy)
}

func filterByPolicy(v any, policy FieldPolicy) any {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	// Handle PageResponse[T] — iterate items
	var pr map[string]json.RawMessage
	if err := json.Unmarshal(data, &pr); err == nil {
		if items, ok := pr["items"]; ok {
			var arr []json.RawMessage
			if err := json.Unmarshal(items, &arr); err == nil {
				filtered := make([]any, len(arr))
				for i, item := range arr {
					filtered[i] = filterMap(item, policy)
				}
				pr["items"], _ = json.Marshal(filtered)
				return pr
			}
		}
	}
	return filterMap(data, policy)
}

func filterMap(data []byte, policy FieldPolicy) any {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return m
	}
	switch policy.Mode {
	case "allow":
		allowed := map[string]struct{}{}
		for _, f := range policy.Fields {
			allowed[f] = struct{}{}
		}
		for k := range m {
			if _, ok := allowed[k]; !ok {
				delete(m, k)
			}
		}
	case "deny":
		denied := map[string]struct{}{}
		for _, f := range policy.Fields {
			denied[f] = struct{}{}
		}
		for k := range m {
			if _, ok := denied[k]; ok {
				delete(m, k)
			}
		}
	}
	return m
}

// Compile-time: suppress unused import warning
var _ = database.BaseReadModel{}

func ValidateInputFields(c echo.Context, body map[string]any) []string {
	policy, ok := c.Get("rbac_field_policy").(FieldPolicy)
	if !ok || policy.Mode == "all" {
		return nil
	}
	var disallowed []string
	switch policy.Mode {
	case "allow":
		allowed := map[string]struct{}{}
		for _, f := range policy.Fields {
			allowed[f] = struct{}{}
		}
		for k := range body {
			if _, ok := allowed[k]; !ok {
				disallowed = append(disallowed, k)
			}
		}
	case "deny":
		denied := map[string]struct{}{}
		for _, f := range policy.Fields {
			denied[f] = struct{}{}
		}
		for k := range body {
			if _, ok := denied[k]; ok {
				disallowed = append(disallowed, k)
			}
		}
	}
	return disallowed
}
