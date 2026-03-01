package featureflag

import "github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"

func init() {
	rbac.RegisterModule(rbac.ModuleDefinition{
		Name:   "featureflag",
		Fields: []string{"id", "key", "enabled", "description", "metadata", "created_at", "created_by", "updated_at", "updated_by"},
		DefaultPerms: []rbac.Permission{
			{Module: "featureflag", Action: "manage", Fields: rbac.FieldPolicy{Mode: "all"}},
		},
	})
}
