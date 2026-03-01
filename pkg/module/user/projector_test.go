package user

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func TestUserProjector_UserCreated_InsertsRow(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)
	ev := &UserCreated{
		BaseEvent: eventstore.NewBaseEvent("1001", "user", "user.created", 1,
			map[string]string{"user_id": "actor_1"}),
		Email:    "alice@example.com",
		PassHash: "hashed",
	}

	// SQL: VALUES ($1, $2, $3, true, $4, $5, $4, $5, false) → 5 unique positional args
	mock.ExpectExec(`INSERT INTO users_read`).
		WithArgs(int64(1001), ev.Email, ev.PassHash, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, p.Handle(context.Background(), ev))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserProjector_AuditFields_CreatedByFromMetadata(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)
	ev := &UserCreated{
		BaseEvent: eventstore.NewBaseEvent("1002", "user", "user.created", 1,
			map[string]string{"user_id": "seeder"}),
		Email:    "b@b.com",
		PassHash: "h",
	}

	// $4=ts, $5=by (="seeder") — 5 unique positional args in the INSERT
	mock.ExpectExec(`INSERT INTO users_read`).
		WithArgs(int64(1002), "b@b.com", "h", sqlmock.AnyArg(), "seeder").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, p.Handle(context.Background(), ev))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserProjector_EmailChanged_UpdatesRow(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)
	ev := &EmailChanged{
		BaseEvent: eventstore.NewBaseEvent("1001", "user", "user.email_changed", 2,
			map[string]string{"user_id": "actor_1"}),
		OldEmail: "old@example.com",
		NewEmail: "new@example.com",
	}

	mock.ExpectExec(`UPDATE users_read SET email`).
		WithArgs(ev.NewEmail, sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1001)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, p.Handle(context.Background(), ev))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserProjector_UserDeleted_SetsIsDeleted(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)
	ev := &UserDeleted{
		BaseEvent: eventstore.NewBaseEvent("1001", "user", "user.deleted", 3,
			map[string]string{"user_id": "actor_1"}),
	}

	mock.ExpectExec(`UPDATE users_read SET is_deleted=true`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1001)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, p.Handle(context.Background(), ev))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserProjector_RoleAssigned_InsertsUserRole(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)
	ev := &RoleAssigned{
		BaseEvent: eventstore.NewBaseEvent("1001", "user", "user.role_assigned", 2,
			map[string]string{"user_id": "actor_1"}),
		RoleID: "2002",
	}

	mock.ExpectExec(`INSERT INTO user_roles_read`).
		WithArgs(int64(1001), int64(2002), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, p.Handle(context.Background(), ev))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserProjector_RoleUnassigned_SetsIsDeleted(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)
	ev := &RoleUnassigned{
		BaseEvent: eventstore.NewBaseEvent("1001", "user", "user.role_unassigned", 3,
			map[string]string{"user_id": "actor_1"}),
		RoleID: "2002",
	}

	mock.ExpectExec(`UPDATE user_roles_read SET is_deleted=true`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1001), int64(2002)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, p.Handle(context.Background(), ev))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserProjector_UnknownEvent_NoOp(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	p := NewProjector(db)
	ev := eventstore.NewBaseEvent("agg_1", "user", "user.unknown", 1, nil)
	assert.NoError(t, p.Handle(context.Background(), &ev))
}

func TestUserProjector_Name(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	assert.Equal(t, "user", NewProjector(db).Name())
}
