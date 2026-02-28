package observability

import (
	"net/http"

	ddhttp "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
)

// NewHTTPClient returns a *http.Client with Datadog APM tracing on all requests.
// The service name is inherited from the global tracer started by Init().
// Inject as a dependency wherever outbound HTTP calls are needed.
func NewHTTPClient() *http.Client {
	return ddhttp.WrapClient(&http.Client{})
}
