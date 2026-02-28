package eventstore

import (
	"encoding/json"
	"fmt"
)

type factory func(data []byte) (Event, error)

var registry = map[string]factory{}

// Register maps an event type string to a concrete Go type T.
func Register[T Event](eventType string) {
	registry[eventType] = func(data []byte) (Event, error) {
		var ev T
		if err := json.Unmarshal(data, &ev); err != nil {
			return nil, fmt.Errorf("eventstore: decode %s: %w", eventType, err)
		}
		return ev, nil
	}
}

func Deserialise(eventType string, data []byte) (Event, error) {
	f, ok := registry[eventType]
	if !ok {
		return nil, fmt.Errorf("eventstore: unknown event type %q", eventType)
	}
	return f(data)
}
