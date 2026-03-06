package envvar

import "github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"

func init() {
	rbac.RegisterModule(rbac.ModuleDefinition{
		Name:   "envvar",
		Fields: []string{"id", "platform", "key", "value", "created_at", "created_by", "updated_at", "updated_by"},
		DefaultPerms: []rbac.Permission{
			{Module: "envvar", Action: "manage", Fields: rbac.FieldPolicy{Mode: "all"}},
		},
	})
}
