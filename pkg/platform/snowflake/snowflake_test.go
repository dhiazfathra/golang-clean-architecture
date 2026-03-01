package snowflake

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewID(t *testing.T) {
	id1 := NewID()
	id2 := NewID()
	assert.NotZero(t, id1)
	assert.NotZero(t, id2)
	assert.NotEqual(t, id1, id2, "two consecutive IDs must be unique")
}

func TestNewStringID(t *testing.T) {
	s := NewStringID()
	assert.NotEmpty(t, s)
}

func TestGetNodeDefault(t *testing.T) {
	n := getNode()
	assert.NotNil(t, n)
}

func TestResolveNodeIDDefault(t *testing.T) {
	t.Setenv("SNOWFLAKE_NODE_ID", "")
	assert.Equal(t, int64(1), resolveNodeID())
}

func TestResolveNodeIDFromEnv(t *testing.T) {
	t.Setenv("SNOWFLAKE_NODE_ID", "42")
	assert.Equal(t, int64(42), resolveNodeID())
}

func TestResolveNodeIDInvalidFallback(t *testing.T) {
	t.Setenv("SNOWFLAKE_NODE_ID", "not-a-number")
	assert.Equal(t, int64(1), resolveNodeID())
}
