package eventstore

import "context"

type EventStore interface {
	Append(ctx context.Context, events []Event) error
	Load(ctx context.Context, aggregateType, aggregateID string, fromVersion int) ([]Event, error)
}
