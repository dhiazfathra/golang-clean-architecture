package eventstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetRegistry(t *testing.T) {
	t.Helper()
	original := registry
	t.Cleanup(func() { registry = original })
	registry = map[string]factory{}
}

func TestDeserialise_UnknownEventType(t *testing.T) {
	resetRegistry(t)

	_, err := Deserialise("GhostEvent", []byte(`{}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown event type`)
	assert.Contains(t, err.Error(), `GhostEvent`)
}
