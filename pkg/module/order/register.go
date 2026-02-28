package order

import (
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func init() {
	eventstore.Register[*OrderCreated]("order.created")
	eventstore.Register[*OrderUpdated]("order.updated")
	eventstore.Register[*OrderDeleted]("order.deleted")

	rbac.RegisterModule(rbac.ModuleDefinition{
		Name:         "order",
		Fields:       []string{"id", "user_id", "status", "total", "created_at", "created_by", "updated_at", "updated_by"},
		DefaultPerms: rbac.FullCRUD("order", rbac.AllFields()),
	})
}
