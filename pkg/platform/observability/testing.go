package observability

import "github.com/DataDog/datadog-go/v5/statsd"

// InitNoop initializes no-op stubs for all observability components.
// Call in TestMain or test helper setup — prevents "no agent" errors in tests.
func InitNoop() {
	statsdClient = &statsd.NoOpClient{}
}
