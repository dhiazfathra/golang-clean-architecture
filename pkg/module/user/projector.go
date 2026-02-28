package user

import (
	"context"
	"strconv"

	"github.com/jmoiron/sqlx"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
)

type Projector struct{ db *sqlx.DB }

func NewProjector(db *sqlx.DB) *Projector { return &Projector{db: db} }

func (p *Projector) Name() string { return "user" }

func (p *Projector) Handle(ctx context.Context, e eventstore.Event) error {
	at := e.Timestamp()
	by := e.Metadata()["user_id"]
	switch ev := e.(type) {
	case *UserCreated:
		userID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO users_read (id, email, pass_hash, active, created_at, created_by, updated_at, updated_by, is_deleted)
			VALUES ($1, $2, $3, true, $4, $5, $4, $5, false)
			ON CONFLICT (id) DO UPDATE SET email=$2, pass_hash=$3, updated_at=$4, updated_by=$5`,
			userID, ev.Email, ev.PassHash, at, by)
		return err
	case *EmailChanged:
		userID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		_, err := p.db.ExecContext(ctx, `
			UPDATE users_read SET email=$1, updated_at=$2, updated_by=$3 WHERE id=$4`,
			ev.NewEmail, at, by, userID)
		return err
	case *UserDeleted:
		userID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		_, err := p.db.ExecContext(ctx, `
			UPDATE users_read SET is_deleted=true, active=false, updated_at=$1, updated_by=$2 WHERE id=$3`,
			at, by, userID)
		return err
	case *RoleAssigned:
		userID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		roleID, _ := strconv.ParseInt(ev.RoleID, 10, 64)
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO user_roles_read (user_id, role_id, created_at, created_by, updated_at, updated_by, is_deleted)
			VALUES ($1, $2, $3, $4, $3, $4, false)
			ON CONFLICT (user_id, role_id) DO UPDATE SET is_deleted=false, updated_at=$3, updated_by=$4`,
			userID, roleID, at, by)
		return err
	case *RoleUnassigned:
		userID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		roleID, _ := strconv.ParseInt(ev.RoleID, 10, 64)
		_, err := p.db.ExecContext(ctx, `
			UPDATE user_roles_read SET is_deleted=true, updated_at=$1, updated_by=$2
			WHERE user_id=$3 AND role_id=$4`,
			at, by, userID, roleID)
		return err
	}
	return nil
}
