package eventstore

import (
	"context"
)

// MockEventStore is a no-op EventStore for use in route/unit tests.
type MockEventStore struct{}

func (m *MockEventStore) Append(_ context.Context, _ []Event) error {
	return nil
}
func (m *MockEventStore) Load(_ context.Context, _, _ string, _ int) ([]Event, error) {
	return nil, nil
}
