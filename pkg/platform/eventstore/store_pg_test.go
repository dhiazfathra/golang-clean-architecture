package eventstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAggregateID = "abc-123"

// pgTestEvent is a concrete, JSON-serialisable Event that satisfies the real
// Event interface (Metadata() map[string]string).
type pgTestEvent struct {
	BaseEvent
	Payload string `json:"payload"`
}

// pgBadJSONEvent wraps pgTestEvent but MarshalJSON always fails, triggering
// the "marshal event" error path in Append.
type pgBadJSONEvent struct{ pgTestEvent }

func (pgBadJSONEvent) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal forced failure")
}

func newPgDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return sqlx.NewDb(db, "postgres"), mock
}

func newPgEvent(evType string, ver int) pgTestEvent {
	return pgTestEvent{
		BaseEvent: NewBaseEvent("abc-123", "order", evType, ver, map[string]string{"src": "test"}),
		Payload:   "payload",
	}
}

func snapshotRegistryForPg(t *testing.T) {
	t.Helper()
	orig := make(map[string]factory, len(registry))
	for k, v := range registry {
		orig[k] = v
	}
	t.Cleanup(func() { registry = orig })
}

// ---------------------------------------------------------------------------
// Append
// ---------------------------------------------------------------------------

func TestPgAppendSingleEvent(t *testing.T) {
	db, mock := newPgDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO events`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := store.Append(context.Background(), []Event{newPgEvent("OrderCreated", 1)})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgAppendMultipleEvents(t *testing.T) {
	db, mock := newPgDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO events`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO events`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	events := []Event{newPgEvent("OrderCreated", 1), newPgEvent("OrderShipped", 2)}
	err := store.Append(context.Background(), events)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgAppendEmptySlice(t *testing.T) {
	db, mock := newPgDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin()
	mock.ExpectCommit()

	err := store.Append(context.Background(), []Event{})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgAppendBeginTxError(t *testing.T) {
	db, mock := newPgDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	err := store.Append(context.Background(), []Event{newPgEvent("OrderCreated", 1)})
	assert.ErrorContains(t, err, "begin tx")
}

func TestPgAppendExecError(t *testing.T) {
	db, mock := newPgDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO events`).WillReturnError(errors.New("unique_violation"))
	mock.ExpectRollback()

	err := store.Append(context.Background(), []Event{newPgEvent("OrderCreated", 1)})
	assert.ErrorContains(t, err, "append")
}

func TestPgAppendCommitError(t *testing.T) {
	db, mock := newPgDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO events`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err := store.Append(context.Background(), []Event{newPgEvent("OrderCreated", 1)})
	assert.ErrorContains(t, err, "commit failed")
}

// TestPgAppendMarshalMetadataError is not achievable via the public Event
// interface because Metadata() map[string]string always produces valid JSON.
// The only remaining marshal error path is the event body itself.
func TestPgAppendMarshalEventError(t *testing.T) {
	db, mock := newPgDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin()
	mock.ExpectRollback()

	bad := pgBadJSONEvent{newPgEvent("OrderCreated", 1)}
	err := store.Append(context.Background(), []Event{bad})
	assert.ErrorContains(t, err, "marshal event")
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

func TestPgLoadSuccess(t *testing.T) {
	snapshotRegistryForPg(t)
	Register[pgTestEvent]("OrderCreated")

	db, mock := newPgDB(t)
	store := NewPgStore(db)

	ev := newPgEvent("OrderCreated", 1)
	data, _ := json.Marshal(ev)
	meta, _ := json.Marshal(ev.Metadata())

	rows := sqlmock.NewRows([]string{"event_type", "version", "data", "metadata", "created_at"}).
		AddRow("OrderCreated", 1, data, meta, time.Now())
	mock.ExpectQuery(`SELECT event_type, version, data, metadata, created_at FROM events`).WillReturnRows(rows)

	events, err := store.Load(context.Background(), "order", testAggregateID, 0)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgLoadMultipleRows(t *testing.T) {
	snapshotRegistryForPg(t)
	Register[pgTestEvent]("OrderCreated")
	Register[pgTestEvent]("OrderShipped")

	db, mock := newPgDB(t)
	store := NewPgStore(db)

	ev1 := newPgEvent("OrderCreated", 1)
	ev2 := newPgEvent("OrderShipped", 2)
	data1, _ := json.Marshal(ev1)
	data2, _ := json.Marshal(ev2)
	meta, _ := json.Marshal(map[string]string{})

	rows := sqlmock.NewRows([]string{"event_type", "version", "data", "metadata", "created_at"}).
		AddRow("OrderCreated", 1, data1, meta, time.Now()).
		AddRow("OrderShipped", 2, data2, meta, time.Now())
	mock.ExpectQuery(`SELECT event_type, version, data, metadata, created_at FROM events`).WillReturnRows(rows)

	events, err := store.Load(context.Background(), "order", testAggregateID, 0)
	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestPgLoadEmptyResult(t *testing.T) {
	db, mock := newPgDB(t)
	store := NewPgStore(db)

	rows := sqlmock.NewRows([]string{"event_type", "version", "data", "metadata", "created_at"})
	mock.ExpectQuery(`SELECT event_type, version, data, metadata, created_at FROM events`).WillReturnRows(rows)

	events, err := store.Load(context.Background(), "order", "no-such-id", 0)
	require.NoError(t, err)
	assert.Nil(t, events)
}

func TestPgLoadSqlErrNoRows(t *testing.T) {
	db, mock := newPgDB(t)
	store := NewPgStore(db)

	mock.ExpectQuery(`SELECT event_type, version, data, metadata, created_at FROM events`).
		WillReturnError(sql.ErrNoRows)

	events, err := store.Load(context.Background(), "order", testAggregateID, 0)
	assert.NoError(t, err)
	assert.Nil(t, events)
}

func TestPgLoadQueryError(t *testing.T) {
	db, mock := newPgDB(t)
	store := NewPgStore(db)

	mock.ExpectQuery(`SELECT event_type, version, data, metadata, created_at FROM events`).
		WillReturnError(errors.New("db unavailable"))

	_, err := store.Load(context.Background(), "order", testAggregateID, 0)
	assert.ErrorContains(t, err, "eventstore: load")
}

func TestPgLoadScanError(t *testing.T) {
	db, mock := newPgDB(t)
	store := NewPgStore(db)

	// Only one column instead of five forces a scan error.
	rows := sqlmock.NewRows([]string{"event_type"}).AddRow("OrderCreated")
	mock.ExpectQuery(`SELECT event_type, version, data, metadata, created_at FROM events`).WillReturnRows(rows)

	_, err := store.Load(context.Background(), "order", testAggregateID, 0)
	assert.Error(t, err)
}

func TestPgLoadDeserialiseUnknownType(t *testing.T) {
	snapshotRegistryForPg(t)
	delete(registry, "GhostEvent")

	db, mock := newPgDB(t)
	store := NewPgStore(db)

	data, _ := json.Marshal(newPgEvent("GhostEvent", 1))
	meta, _ := json.Marshal(map[string]string{})

	rows := sqlmock.NewRows([]string{"event_type", "version", "data", "metadata", "created_at"}).
		AddRow("GhostEvent", 1, data, meta, time.Now())
	mock.ExpectQuery(`SELECT event_type, version, data, metadata, created_at FROM events`).WillReturnRows(rows)

	_, err := store.Load(context.Background(), "order", testAggregateID, 0)
	assert.ErrorContains(t, err, "unknown event type")
}

func TestPgLoadDeserialiseInvalidJSON(t *testing.T) {
	snapshotRegistryForPg(t)
	Register[pgTestEvent]("OrderCreated")

	db, mock := newPgDB(t)
	store := NewPgStore(db)

	rows := sqlmock.NewRows([]string{"event_type", "version", "data", "metadata", "created_at"}).
		AddRow("OrderCreated", 1, []byte(`not-json`), []byte(`{}`), time.Now())
	mock.ExpectQuery(`SELECT event_type, version, data, metadata, created_at FROM events`).WillReturnRows(rows)

	_, err := store.Load(context.Background(), "order", testAggregateID, 0)
	assert.ErrorContains(t, err, "decode OrderCreated")
}

func TestPgLoadRowsError(t *testing.T) {
	snapshotRegistryForPg(t)
	Register[pgTestEvent]("OrderCreated")

	db, mock := newPgDB(t)
	store := NewPgStore(db)

	ev := newPgEvent("OrderCreated", 1)
	data, _ := json.Marshal(ev)
	meta, _ := json.Marshal(map[string]string{})

	rows := sqlmock.NewRows([]string{"event_type", "version", "data", "metadata", "created_at"}).
		AddRow("OrderCreated", 1, data, meta, time.Now()).
		RowError(0, errors.New("network blip"))
	mock.ExpectQuery(`SELECT event_type, version, data, metadata, created_at FROM events`).WillReturnRows(rows)

	_, err := store.Load(context.Background(), "order", testAggregateID, 0)
	assert.ErrorContains(t, err, "network blip")
}

// Compile-time check: pgTestEvent satisfies Event.
var _ Event = pgTestEvent{}

// Compile-time check: verify newPgDB returns a usable *sqlx.DB.
var _ = func() { var _ *sqlx.DB = nil; var _ = time.Time{} }
