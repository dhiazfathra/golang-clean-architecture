package snowflake

import (
	"os"
	"strconv"
	"sync"

	bwsnowflake "github.com/bwmarrin/snowflake"
)

var (
	once sync.Once
	node *bwsnowflake.Node
)

// resolveNodeID reads SNOWFLAKE_NODE_ID from the environment, falling back to 1.
func resolveNodeID() int64 {
	if v := os.Getenv("SNOWFLAKE_NODE_ID"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return 1
}

func getNode() *bwsnowflake.Node {
	once.Do(func() {
		var err error
		node, err = bwsnowflake.NewNode(resolveNodeID())
		if err != nil {
			panic("snowflake: failed to create node: " + err.Error())
		}
	})
	return node
}

// NewID returns a new unique Snowflake int64 ID.
func NewID() int64 {
	return getNode().Generate().Int64()
}

// NewStringID returns a new unique Snowflake ID as its decimal string representation.
// Use this when the ID must cross the event-store boundary (aggregate_id TEXT).
func NewStringID() string {
	return strconv.FormatInt(NewID(), 10)
}
