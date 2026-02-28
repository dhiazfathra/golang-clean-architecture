package user

import (
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func init() {
	eventstore.Register[*UserCreated]("user.created")
	eventstore.Register[*EmailChanged]("user.email_changed")
	eventstore.Register[*UserDeleted]("user.deleted")
	eventstore.Register[*RoleAssigned]("user.role_assigned")
	eventstore.Register[*RoleUnassigned]("user.role_unassigned")

	rbac.RegisterModule(rbac.ModuleDefinition{
		Name:         "user",
		Fields:       []string{"id", "email", "active", "created_at", "created_by", "updated_at", "updated_by"},
		DefaultPerms: rbac.FullCRUD("user", rbac.AllFields()),
	})
}
