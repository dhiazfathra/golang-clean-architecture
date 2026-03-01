package order

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func TestOrderProjector_OrderCreated_InsertsRow(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)
	ev := &OrderCreated{
		BaseEvent: eventstore.NewBaseEvent("2001", "order", "order.created", 1,
			map[string]string{"user_id": "actor_1"}),
		UserID: "1001",
		Total:  49.99,
	}

	// SQL: VALUES ($1, $2, 'pending', $3, $4, $5, $4, $5, false) → 5 unique positional args
	mock.ExpectExec(`INSERT INTO orders_read`).
		WithArgs(int64(2001), int64(1001), ev.Total, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, p.Handle(context.Background(), ev))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderProjector_OrderUpdated_UpdatesRow(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)
	ev := &OrderUpdated{
		BaseEvent: eventstore.NewBaseEvent("2001", "order", "order.updated", 2,
			map[string]string{"user_id": "actor_1"}),
		Status: "completed",
		Total:  99.0,
	}

	mock.ExpectExec(`UPDATE orders_read SET status`).
		WithArgs("completed", 99.0, sqlmock.AnyArg(), sqlmock.AnyArg(), int64(2001)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, p.Handle(context.Background(), ev))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderProjector_OrderDeleted_SetsIsDeleted(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)
	ev := &OrderDeleted{
		BaseEvent: eventstore.NewBaseEvent("2001", "order", "order.deleted", 3,
			map[string]string{"user_id": "actor_1"}),
	}

	mock.ExpectExec(`UPDATE orders_read SET is_deleted=true`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(2001)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, p.Handle(context.Background(), ev))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderProjector_UnknownEvent_NoOp(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	p := NewProjector(db)
	ev := eventstore.NewBaseEvent("agg_1", "order", "order.unknown", 1, nil)
	assert.NoError(t, p.Handle(context.Background(), &ev))
}

func TestOrderProjector_Name(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	assert.Equal(t, "order", NewProjector(db).Name())
}
