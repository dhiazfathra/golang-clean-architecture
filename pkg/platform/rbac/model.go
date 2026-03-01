package rbac

import (
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
)

// --- Permission model ---

type Permission struct {
	Module string      `json:"module"`
	Action string      `json:"action"`
	Fields FieldPolicy `json:"fields"`
}

type FieldPolicy struct {
	Mode   string   `json:"mode"` // "all" | "allow" | "deny"
	Fields []string `json:"fields,omitempty"`
}

func AllFields() FieldPolicy              { return FieldPolicy{Mode: "all"} }
func AllowFields(f ...string) FieldPolicy { return FieldPolicy{Mode: "allow", Fields: f} }
func DenyFields(f ...string) FieldPolicy  { return FieldPolicy{Mode: "deny", Fields: f} }

func FullCRUD(module string, fp FieldPolicy) []Permission {
	actions := []string{"create", "read", "update", "delete", "list"}
	perms := make([]Permission, len(actions))
	for i, a := range actions {
		perms[i] = Permission{Module: module, Action: a, Fields: fp}
	}
	return perms
}

func SuperAdminPermission() Permission {
	return Permission{Module: "*", Action: "*", Fields: AllFields()}
}

// --- Role aggregate ---

type RoleState struct {
	ID          string
	Name        string
	Description string
	Permissions []Permission
	Active      bool
}

type RoleCreated struct {
	eventstore.BaseEvent
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
}

type PermissionGranted struct {
	eventstore.BaseEvent
	Permission Permission `json:"permission"`
}

type PermissionRevoked struct {
	eventstore.BaseEvent
	Module string `json:"module"`
	Action string `json:"action"`
}

type RoleDeleted struct {
	eventstore.BaseEvent
}

func applyRole(s *RoleState, e eventstore.Event) {
	switch ev := e.(type) {
	case *RoleCreated:
		s.ID = ev.AggregateID()
		s.Name = ev.Name
		s.Description = ev.Description
		s.Permissions = ev.Permissions
		s.Active = true
	case *PermissionGranted:
		s.Permissions = append(s.Permissions, ev.Permission)
	case *PermissionRevoked:
		kept := s.Permissions[:0]
		for _, p := range s.Permissions {
			if p.Module != ev.Module || p.Action != ev.Action {
				kept = append(kept, p)
			}
		}
		s.Permissions = kept
	case *RoleDeleted:
		s.Active = false
	}
}

func newRoleAggregate(id string) *eventstore.Aggregate[RoleState] {
	return eventstore.New[RoleState](id, applyRole)
}
