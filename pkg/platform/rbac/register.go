package rbac

import "github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"

func init() {
	eventstore.Register[*RoleCreated]("role.created")
	eventstore.Register[*PermissionGranted]("role.permission_granted")
	eventstore.Register[*PermissionRevoked]("role.permission_revoked")
	eventstore.Register[*RoleDeleted]("role.deleted")
}
