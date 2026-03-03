package main

import (
	"testing"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/config"
	"github.com/stretchr/testify/assert"
)

func TestSetupRouter(t *testing.T) {
	// Minimal deps - we still use nil because we won't execute the handlers
	deps := RouterDeps{
		Cfg: config.Config{
			ServiceName: "test-service",
			Env:         "development",
		},
	}

	t.Run("should register all expected routes", func(t *testing.T) {
		e := setupRouter(deps)
		routes := e.Routes()

		// Helper to check if a route exists in the registry
		hasRoute := func(method, path string) bool {
			for _, r := range routes {
				if r.Method == method && r.Path == path {
					return true
				}
			}
			return false
		}

		assert.True(t, hasRoute("GET", "/health"), "Missing /health")
		assert.True(t, hasRoute("GET", "/health/ready"), "Missing /health/ready")
		assert.True(t, hasRoute("GET", "/docs"), "Missing /docs in development")
	})

	t.Run("should respect environment flags for docs", func(t *testing.T) {
		// Test Production: Docs should be missing
		deps.Cfg.Env = "production"
		eProd := setupRouter(deps)

		for _, r := range eProd.Routes() {
			assert.NotEqual(t, "/docs", r.Path, "Docs should not be registered in production")
		}

		// Test Development: Docs should be present
		deps.Cfg.Env = "development"
		eDev := setupRouter(deps)
		found := false
		for _, r := range eDev.Routes() {
			if r.Path == "/docs" {
				found = true
				break
			}
		}
		assert.True(t, found, "Docs should be registered in development")
	})
}
