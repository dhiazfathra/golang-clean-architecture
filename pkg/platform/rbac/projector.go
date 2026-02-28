package rbac

import (
	"context"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/snowflake"
)

type Projector struct{ db *sqlx.DB }

func NewProjector(db *sqlx.DB) *Projector { return &Projector{db: db} }

func (p *Projector) Name() string { return "rbac" }

func (p *Projector) Handle(ctx context.Context, e eventstore.Event) error {
	at, by := e.Timestamp(), e.Metadata()["user_id"]
	switch ev := e.(type) {
	case *RoleCreated:
		roleID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO roles_read (id, name, description, created_at, created_by, updated_at, updated_by, is_deleted)
			VALUES ($1,$2,$3,$4,$5,$4,$5,false)
			ON CONFLICT (id) DO UPDATE SET name=$2, description=$3, updated_at=$4, updated_by=$5`,
			roleID, ev.Name, ev.Description, at, by)
		return err
	case *PermissionGranted:
		roleID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		permID := snowflake.NewID()
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO permissions_read (id,role_id,module,action,field_mode,field_list,created_at,created_by,updated_at,updated_by,is_deleted)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$7,$8,false)
			ON CONFLICT (role_id, module, action)
			DO UPDATE SET field_mode=$5,field_list=$6,is_deleted=false,updated_at=$7,updated_by=$8`,
			permID, roleID, ev.Permission.Module, ev.Permission.Action,
			ev.Permission.Fields.Mode, pq.Array(ev.Permission.Fields.Fields), at, by)
		return err
	case *PermissionRevoked:
		roleID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		_, err := p.db.ExecContext(ctx, `
			UPDATE permissions_read SET is_deleted=true, updated_at=$1, updated_by=$2
			WHERE role_id=$3 AND module=$4 AND action=$5`,
			at, by, roleID, ev.Module, ev.Action)
		return err
	case *RoleDeleted:
		roleID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		_, err := p.db.ExecContext(ctx, `
			UPDATE roles_read SET is_deleted=true, updated_at=$1, updated_by=$2 WHERE id=$3`,
			at, by, roleID)
		return err
	}
	return nil
}
