package rbac

import (
	"github.com/lib/pq"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
)

type RoleReadModel struct {
	ID          int64  `db:"id"          json:"id"`
	Name        string `db:"name"        json:"name"`
	Description string `db:"description" json:"description"`
	database.BaseReadModel
}

type PermissionReadModel struct {
	ID        int64          `db:"id"         json:"id"`
	RoleID    int64          `db:"role_id"    json:"role_id"`
	Module    string         `db:"module"     json:"module"`
	Action    string         `db:"action"     json:"action"`
	FieldMode string         `db:"field_mode" json:"field_mode"`
	FieldList pq.StringArray `db:"field_list" json:"field_list"`
	database.BaseReadModel
}

type UserRoleReadModel struct {
	UserID int64 `db:"user_id" json:"user_id"`
	RoleID int64 `db:"role_id" json:"role_id"`
	database.BaseReadModel
}
