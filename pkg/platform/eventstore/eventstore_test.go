package eventstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/logging"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/observability"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func init() {
	observability.InitNoop()
}

const (
	qSelectEvents    = "SELECT event_type, version, data, metadata, created_at FROM events"
	qSelectAllEvents = "SELECT id, aggregate_type, aggregate_id, event_type, version, data, metadata, created_at"
	qSelectCursor    = "SELECT last_event_id FROM projection_cursors"
)

// --- BaseEvent ---

func TestNewBaseEvent(t *testing.T) {
	meta := map[string]string{"user": "alice"}
	e := NewBaseEvent("agg1", "Order", "OrderCreated", 1, meta)

	assert.Equal(t, "agg1", e.AggregateID())
	assert.Equal(t, "Order", e.AggregateType())
	assert.Equal(t, "OrderCreated", e.EventType())
	assert.Equal(t, 1, e.Version())
	assert.WithinDuration(t, time.Now().UTC(), e.Timestamp(), 2*time.Second)
	assert.Equal(t, meta, e.Metadata())
}

func TestBaseEventNilMetadata(t *testing.T) {
	e := NewBaseEvent("a", "T", "E", 0, nil)
	assert.Nil(t, e.Metadata())
}

// --- Aggregate ---

type testState struct {
	Count int
}

func testApply(s *testState, e Event) {
	if e.EventType() == "Increment" {
		s.Count++
	}
}

func TestAggregateNew(t *testing.T) {
	agg := New("id1", testApply)
	assert.Equal(t, "id1", agg.ID)
	assert.Equal(t, 0, agg.Version)
	assert.Equal(t, testState{}, agg.State)
	assert.Empty(t, agg.Uncommitted())
}

func TestAggregateApply(t *testing.T) {
	agg := New("id1", testApply)
	e := NewBaseEvent("id1", "Test", "Increment", 1, nil)

	agg.Apply(&e)

	assert.Equal(t, 1, agg.State.Count)
	assert.Equal(t, 1, agg.Version)
	assert.Len(t, agg.Uncommitted(), 1)
}

func TestAggregateClearUncommitted(t *testing.T) {
	agg := New("id1", testApply)
	e := NewBaseEvent("id1", "Test", "Increment", 1, nil)
	agg.Apply(&e)
	assert.Len(t, agg.Uncommitted(), 1)

	agg.ClearUncommitted()
	assert.Empty(t, agg.Uncommitted())
}

func TestAggregateRehydrate(t *testing.T) {
	agg := New("id1", testApply)
	e := NewBaseEvent("id1", "Test", "Increment", 5, nil)

	agg.Rehydrate(&e)

	assert.Equal(t, 1, agg.State.Count)
	assert.Equal(t, 5, agg.Version)
	assert.Empty(t, agg.Uncommitted())
}

func TestAggregateMultipleApply(t *testing.T) {
	agg := New("id1", testApply)
	for i := 1; i <= 3; i++ {
		e := NewBaseEvent("id1", "Test", "Increment", i, nil)
		agg.Apply(&e)
	}
	assert.Equal(t, 3, agg.State.Count)
	assert.Equal(t, 3, agg.Version)
	assert.Len(t, agg.Uncommitted(), 3)
}

// --- Registry ---

type testEvent struct {
	BaseEvent
	Name string `json:"name"`
}

func TestRegisterAndDeserialise(t *testing.T) {
	Register[*testEvent]("TestEvent")

	data, _ := json.Marshal(&testEvent{Name: "hello"})
	e, err := Deserialise("TestEvent", data)

	require.NoError(t, err)
	te, ok := e.(*testEvent)
	require.True(t, ok)
	assert.Equal(t, "hello", te.Name)
}

func TestDeserialiseUnknownType(t *testing.T) {
	_, err := Deserialise("NoSuchType", []byte(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type")
}

func TestDeserialiseInvalidJSON(t *testing.T) {
	Register[*testEvent]("TestEventBad")
	_, err := Deserialise("TestEventBad", []byte(`{invalid`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

// unmarshalableEvent always fails json.Marshal.
type unmarshalableEvent struct {
	BaseEvent
}

func (u unmarshalableEvent) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal fail")
}

// --- PgStore ---

func TestPgStoreAppend(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO events").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	e := NewBaseEvent("a1", "Order", "OrderCreated", 1, map[string]string{"k": "v"})
	err := store.Append(context.Background(), []Event{&e})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgStoreAppendMarshalEventError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin()
	mock.ExpectRollback()

	e := unmarshalableEvent{BaseEvent: NewBaseEvent("a", "T", "E", 1, nil)}
	err := store.Append(context.Background(), []Event{&e})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal event")
}

func TestPgStoreAppendBeginError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin().WillReturnError(errors.New("begin fail"))

	e := NewBaseEvent("a", "T", "E", 1, nil)
	err := store.Append(context.Background(), []Event{&e})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "begin tx")
}

func TestPgStoreAppendExecError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO events").WillReturnError(errors.New("exec fail"))
	mock.ExpectRollback()

	e := NewBaseEvent("a", "T", "E", 1, nil)
	err := store.Append(context.Background(), []Event{&e})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "append")
}

func TestPgStoreAppendCommitError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO events").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit fail"))

	e := NewBaseEvent("a", "T", "E", 1, nil)
	err := store.Append(context.Background(), []Event{&e})
	assert.Error(t, err)
}

func TestPgStoreAppendEmpty(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	mock.ExpectBegin()
	mock.ExpectCommit()

	err := store.Append(context.Background(), []Event{})
	require.NoError(t, err)
}

func TestPgStoreLoad(t *testing.T) {
	Register[*testEvent]("LoadTest")
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	data, _ := json.Marshal(testEvent{Name: "loaded"})
	meta, _ := json.Marshal(map[string]string{})

	rows := sqlmock.NewRows([]string{"event_type", "version", "data", "metadata", "created_at"}).
		AddRow("LoadTest", 1, data, meta, time.Now())
	mock.ExpectQuery(qSelectEvents).
		WithArgs("Order", "a1", 0).
		WillReturnRows(rows)

	events, err := store.Load(context.Background(), "Order", "a1", 0)
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestPgStoreLoadErrNoRows(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	mock.ExpectQuery(qSelectEvents).
		WithArgs("Order", "a1", 0).
		WillReturnError(sql.ErrNoRows)

	events, err := store.Load(context.Background(), "Order", "a1", 0)
	require.NoError(t, err)
	assert.Nil(t, events)
}

func TestPgStoreLoadQueryError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	mock.ExpectQuery(qSelectEvents).
		WillReturnError(errors.New("query fail"))

	_, err := store.Load(context.Background(), "Order", "a1", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load")
}

func TestPgStoreLoadScanError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	// Use wrong number of columns to trigger scan error
	rows := sqlmock.NewRows([]string{"event_type"}).AddRow("x")
	mock.ExpectQuery(qSelectEvents).
		WithArgs("Order", "a1", 0).
		WillReturnRows(rows)

	_, err := store.Load(context.Background(), "Order", "a1", 0)
	assert.Error(t, err)
}

func TestPgStoreLoadDeserialiseError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	rows := sqlmock.NewRows([]string{"event_type", "version", "data", "metadata", "created_at"}).
		AddRow("UnknownEventType999", 1, []byte(`{}`), []byte(`{}`), time.Now())
	mock.ExpectQuery(qSelectEvents).
		WithArgs("Order", "a1", 0).
		WillReturnRows(rows)

	_, err := store.Load(context.Background(), "Order", "a1", 0)
	assert.Error(t, err)
}

func TestPgStoreLoadRowsErr(t *testing.T) {
	Register[*testEvent]("RowsErrEvent")
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)

	data, _ := json.Marshal(testEvent{Name: "x"})
	rows := sqlmock.NewRows([]string{"event_type", "version", "data", "metadata", "created_at"}).
		AddRow("RowsErrEvent", 1, data, []byte(`{}`), time.Now()).
		RowError(0, errors.New("row iteration error"))

	mock.ExpectQuery(qSelectEvents).
		WithArgs("Order", "a1", 0).
		WillReturnRows(rows)

	_, err := store.Load(context.Background(), "Order", "a1", 0)
	assert.Error(t, err)
}

// --- SnapshotStore ---

func TestNewSnapshotStore(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	s := NewSnapshotStore(db)
	assert.NotNil(t, s)
	assert.Equal(t, defaultSnapshotFrequency, s.frequency)
}

func TestSnapshotSaveSkipsNonMultiple(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	s := NewSnapshotStore(db)

	agg := New("id1", testApply)
	agg.Version = 5

	err := SnapshotSave(context.Background(), s, agg)
	require.NoError(t, err)
}

func TestSnapshotSaveAtFrequency(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	s := NewSnapshotStore(db)

	agg := New("id1", testApply)
	agg.Version = 100

	mock.ExpectExec("INSERT INTO snapshots").WillReturnResult(sqlmock.NewResult(1, 1))

	err := SnapshotSave(context.Background(), s, agg)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSnapshotSaveExecError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	s := NewSnapshotStore(db)

	agg := New("id1", testApply)
	agg.Version = 200

	mock.ExpectExec("INSERT INTO snapshots").WillReturnError(errors.New("db fail"))

	err := SnapshotSave(context.Background(), s, agg)
	assert.Error(t, err)
}

func TestSnapshotSaveMarshalError(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	s := NewSnapshotStore(db)

	// Use a state type that can't be marshaled
	type badState struct {
		Fn func() `json:"fn"`
	}
	agg := New[badState]("id1", func(s *badState, e Event) {})
	agg.Version = 100
	agg.State.Fn = func() {}

	err := SnapshotSave(context.Background(), s, agg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal")
}

func TestSnapshotLoadFound(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	s := NewSnapshotStore(db)

	stateData, _ := json.Marshal(testState{Count: 42})
	mock.ExpectQuery("SELECT version, data FROM snapshots").
		WithArgs("Order", "a1").
		WillReturnRows(sqlmock.NewRows([]string{"version", "data"}).AddRow(100, stateData))

	agg, err := SnapshotLoad[testState](context.Background(), s, "Order", "a1")
	require.NoError(t, err)
	require.NotNil(t, agg)
	assert.Equal(t, 42, agg.State.Count)
	assert.Equal(t, 100, agg.Version)
	assert.Equal(t, "a1", agg.ID)
}

func TestSnapshotLoadNotFound(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	s := NewSnapshotStore(db)

	mock.ExpectQuery("SELECT version, data FROM snapshots").
		WithArgs("Order", "a1").
		WillReturnError(errors.New("no rows"))

	agg, err := SnapshotLoad[testState](context.Background(), s, "Order", "a1")
	require.NoError(t, err)
	assert.Nil(t, agg)
}

func TestSnapshotLoadUnmarshalError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	s := NewSnapshotStore(db)

	mock.ExpectQuery("SELECT version, data FROM snapshots").
		WithArgs("Order", "a1").
		WillReturnRows(sqlmock.NewRows([]string{"version", "data"}).AddRow(100, []byte(`{invalid`)))

	_, err := SnapshotLoad[testState](context.Background(), s, "Order", "a1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

// --- ProjectionRunner ---

type mockProjector struct {
	name    string
	handler func(ctx context.Context, event Event) error
}

func (m *mockProjector) Name() string                                  { return m.name }
func (m *mockProjector) Handle(ctx context.Context, event Event) error { return m.handler(ctx, event) }

func TestNewProjectionRunner(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	store := NewPgStore(db)
	r := NewProjectionRunner(db, store, logging.Noop())
	assert.NotNil(t, r)
	assert.Equal(t, 500*time.Millisecond, r.interval)
}

func pollCols() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "aggregate_type", "aggregate_id", "event_type",
		"version", "data", "metadata", "created_at",
	})
}

func TestProjectionRunnerRegister(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	r := NewProjectionRunner(db, NewPgStore(db), logging.Noop())

	p := &mockProjector{name: "test-proj"}
	r.Register(p)
	assert.Len(t, r.projectors, 1)
}

func TestProjectionRunnerStartAndCancel(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	r := NewProjectionRunner(db, NewPgStore(db), logging.Noop())
	r.interval = 10 * time.Millisecond

	p := &mockProjector{name: "test-proj", handler: func(_ context.Context, _ Event) error { return nil }}
	r.Register(p)

	ctx, cancel := context.WithCancel(context.Background())
	r.Start(ctx)

	time.Sleep(30 * time.Millisecond)
	cancel()
}

func TestProjectionRunnerRunOnceSuccess(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	r := NewProjectionRunner(db, NewPgStore(db), logging.Noop())

	p := &mockProjector{name: "ro-proj", handler: func(_ context.Context, _ Event) error { return nil }}
	r.Register(p)

	mock.ExpectQuery(qSelectCursor).
		WillReturnRows(sqlmock.NewRows([]string{"last_event_id"}).AddRow(0))
	mock.ExpectQuery(qSelectAllEvents).
		WillReturnRows(pollCols())
	mock.ExpectExec("UPDATE projection_cursors").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := r.RunOnce(context.Background())
	require.NoError(t, err)
}

func TestProjectionRunnerRunOnceError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	r := NewProjectionRunner(db, NewPgStore(db), logging.Noop())

	p := &mockProjector{name: "ro-err-proj", handler: func(_ context.Context, _ Event) error { return nil }}
	r.Register(p)

	mock.ExpectQuery(qSelectCursor).
		WillReturnError(errors.New("cursor fail"))
	mock.ExpectExec("INSERT INTO projection_cursors").
		WillReturnError(errors.New("insert fail"))

	err := r.RunOnce(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run once ro-err-proj")
}

func TestProjectionRunnerPollFirstRun(t *testing.T) {
	Register[*testEvent]("PollEvent")
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)
	r := NewProjectionRunner(db, store, logging.Noop())

	handled := false
	p := &mockProjector{
		name: "poll-proj",
		handler: func(_ context.Context, _ Event) error {
			handled = true
			return nil
		},
	}

	mock.ExpectQuery(qSelectCursor).
		WithArgs("poll-proj").
		WillReturnError(errors.New("no rows"))
	mock.ExpectExec("INSERT INTO projection_cursors").
		WillReturnResult(sqlmock.NewResult(1, 1))

	data, _ := json.Marshal(testEvent{Name: "ev1"})
	rows := pollCols().
		AddRow(1, "Order", "a1", "PollEvent", 1, data, []byte(`{}`), time.Now())
	mock.ExpectQuery(qSelectAllEvents).WillReturnRows(rows)

	mock.ExpectExec("UPDATE projection_cursors").
		WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := r.poll(context.Background(), p)
	require.NoError(t, err)
	assert.True(t, handled)
}

func TestProjectionRunnerPollExistingCursor(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)
	r := NewProjectionRunner(db, store, logging.Noop())

	p := &mockProjector{
		name:    "poll-proj2",
		handler: func(_ context.Context, _ Event) error { return nil },
	}

	mock.ExpectQuery(qSelectCursor).
		WithArgs("poll-proj2").
		WillReturnRows(sqlmock.NewRows([]string{"last_event_id"}).AddRow(5))

	mock.ExpectQuery(qSelectAllEvents).
		WillReturnRows(pollCols())

	mock.ExpectExec("UPDATE projection_cursors").
		WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := r.poll(context.Background(), p)
	require.NoError(t, err)
}

func TestProjectionRunnerPollInitCursorError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)
	r := NewProjectionRunner(db, store, logging.Noop())

	p := &mockProjector{name: "fail-proj"}

	mock.ExpectQuery(qSelectCursor).WillReturnError(errors.New("no rows"))
	mock.ExpectExec("INSERT INTO projection_cursors").
		WillReturnError(errors.New("insert fail"))

	_, err := r.poll(context.Background(), p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "init cursor")
}

func TestProjectionRunnerPollQueryError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)
	r := NewProjectionRunner(db, store, logging.Noop())

	p := &mockProjector{name: "qerr-proj"}

	mock.ExpectQuery(qSelectCursor).
		WillReturnRows(sqlmock.NewRows([]string{"last_event_id"}).AddRow(0))

	mock.ExpectQuery(qSelectAllEvents).WillReturnError(errors.New("query fail"))

	_, err := r.poll(context.Background(), p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "poll events")
}

func TestProjectionRunnerPollHandleError(t *testing.T) {
	Register[*testEvent]("HandleErr")
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)
	r := NewProjectionRunner(db, store, logging.Noop())

	p := &mockProjector{
		name:    "herr-proj",
		handler: func(_ context.Context, _ Event) error { return errors.New("handle fail") },
	}

	mock.ExpectQuery(qSelectCursor).
		WillReturnRows(sqlmock.NewRows([]string{"last_event_id"}).AddRow(0))

	data, _ := json.Marshal(testEvent{Name: "x"})
	mock.ExpectQuery(qSelectAllEvents).
		WillReturnRows(pollCols().
			AddRow(1, "Order", "a1", "HandleErr", 1, data, []byte(`{}`), time.Now()))

	_, err := r.poll(context.Background(), p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handle event")
}

func TestProjectionRunnerPollUnknownEventSkipped(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)
	r := NewProjectionRunner(db, store, logging.Noop())

	p := &mockProjector{
		name:    "skip-proj",
		handler: func(_ context.Context, _ Event) error { return nil },
	}

	mock.ExpectQuery(qSelectCursor).
		WillReturnRows(sqlmock.NewRows([]string{"last_event_id"}).AddRow(0))

	mock.ExpectQuery(qSelectAllEvents).
		WillReturnRows(pollCols().
			AddRow(1, "Order", "a1", "CompletelyUnknownEvent999", 1, []byte(`{}`), []byte(`{}`), time.Now()))

	mock.ExpectExec("UPDATE projection_cursors").
		WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := r.poll(context.Background(), p)
	require.NoError(t, err)
}

func TestProjectionRunnerPollScanError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)
	r := NewProjectionRunner(db, store, logging.Noop())

	p := &mockProjector{name: "scan-proj"}

	mock.ExpectQuery(qSelectCursor).
		WillReturnRows(sqlmock.NewRows([]string{"last_event_id"}).AddRow(0))

	mock.ExpectQuery(qSelectAllEvents).
		WillReturnRows(pollCols().
			AddRow("not-an-int", "Order", "a1", "SomeEvent", 1, nil, nil, time.Now()))

	_, err := r.poll(context.Background(), p)
	assert.Error(t, err)
}

func TestProjectionRunnerPollRowsErr(t *testing.T) {
	Register[*testEvent]("PollRowsErr")
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)
	r := NewProjectionRunner(db, store, logging.Noop())

	p := &mockProjector{
		name:    "rerr-proj",
		handler: func(_ context.Context, _ Event) error { return nil },
	}

	mock.ExpectQuery(qSelectCursor).
		WillReturnRows(sqlmock.NewRows([]string{"last_event_id"}).AddRow(0))

	data, _ := json.Marshal(testEvent{Name: "x"})
	rows := pollCols().
		AddRow(1, "Order", "a1", "PollRowsErr", 1, data, []byte(`{}`), time.Now()).
		RowError(0, errors.New("rows iteration fail"))

	mock.ExpectQuery(qSelectAllEvents).WillReturnRows(rows)

	_, err := r.poll(context.Background(), p)
	assert.Error(t, err)
}

func TestProjectionRunnerPollUpdateCursorError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	store := NewPgStore(db)
	r := NewProjectionRunner(db, store, logging.Noop())

	p := &mockProjector{name: "upd-proj", handler: func(_ context.Context, _ Event) error { return nil }}

	mock.ExpectQuery(qSelectCursor).
		WillReturnRows(sqlmock.NewRows([]string{"last_event_id"}).AddRow(0))

	mock.ExpectQuery(qSelectAllEvents).
		WillReturnRows(pollCols())

	mock.ExpectExec("UPDATE projection_cursors").
		WillReturnError(errors.New("update fail"))

	_, err := r.poll(context.Background(), p)
	assert.Error(t, err)
}
