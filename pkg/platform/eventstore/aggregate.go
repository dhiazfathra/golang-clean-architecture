package eventstore

type Aggregate[T any] struct {
	ID          string
	Version     int
	State       T
	uncommitted []Event
	applyFn     func(*T, Event)
}

func New[T any](id string, applyFn func(*T, Event)) *Aggregate[T] {
	return &Aggregate[T]{ID: id, applyFn: applyFn}
}

func (a *Aggregate[T]) Apply(e Event) {
	a.applyFn(&a.State, e)
	a.Version++
	a.uncommitted = append(a.uncommitted, e)
}

func (a *Aggregate[T]) Uncommitted() []Event { return a.uncommitted }

func (a *Aggregate[T]) ClearUncommitted() { a.uncommitted = nil }

// Rehydrate applies a historical event (no tracking — used during load).
func (a *Aggregate[T]) Rehydrate(e Event) {
	a.applyFn(&a.State, e)
	a.Version = e.Version()
}
