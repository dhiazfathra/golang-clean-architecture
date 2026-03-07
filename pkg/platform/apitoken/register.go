package apitoken

import "github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"

func init() {
	rbac.RegisterModule(rbac.ModuleDefinition{
		Name:   "apitoken",
		Fields: []string{"id", "name", "token_prefix", "user_id", "expires_at", "created_at", "created_by", "updated_at", "updated_by"},
		DefaultPerms: []rbac.Permission{
			{Module: "apitoken", Action: "manage", Fields: rbac.FieldPolicy{Mode: "all"}},
		},
	})
}
