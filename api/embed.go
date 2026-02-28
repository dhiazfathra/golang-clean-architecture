// Package api embeds the OpenAPI specification and Scalar UI HTML
// so they are included in the compiled binary.
package api

import "embed"

//go:embed openapi.yaml scalar.html
var Files embed.FS
