package order

import (
	"context"
	"strconv"

	"github.com/jmoiron/sqlx"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
)

type Projector struct{ db *sqlx.DB }

func NewProjector(db *sqlx.DB) *Projector { return &Projector{db: db} }

func (p *Projector) Name() string { return "order" }

func (p *Projector) Handle(ctx context.Context, e eventstore.Event) error {
	at := e.Timestamp()
	by := e.Metadata()["user_id"]
	switch ev := e.(type) {
	case *OrderCreated:
		orderID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		userID, _ := strconv.ParseInt(ev.UserID, 10, 64)
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO orders_read (id, user_id, status, total, created_at, created_by, updated_at, updated_by, is_deleted)
			VALUES ($1, $2, 'pending', $3, $4, $5, $4, $5, false)
			ON CONFLICT (id) DO UPDATE SET user_id=$2, total=$3, updated_at=$4, updated_by=$5`,
			orderID, userID, ev.Total, at, by)
		return err
	case *OrderUpdated:
		orderID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		_, err := p.db.ExecContext(ctx, `
			UPDATE orders_read SET status=$1, total=$2, updated_at=$3, updated_by=$4 WHERE id=$5`,
			ev.Status, ev.Total, at, by, orderID)
		return err
	case *OrderDeleted:
		orderID, _ := strconv.ParseInt(ev.AggregateID(), 10, 64)
		_, err := p.db.ExecContext(ctx, `
			UPDATE orders_read SET is_deleted=true, updated_at=$1, updated_by=$2 WHERE id=$3`,
			at, by, orderID)
		return err
	}
	return nil
}
