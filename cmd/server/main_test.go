package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/config"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// routeSet returns the set of "METHOD /path" strings registered on e.
func routeSet(e *echo.Echo) map[string]bool {
	set := make(map[string]bool)
	for _, r := range e.Routes() {
		set[r.Method+" "+r.Path] = true
	}
	return set
}

// freeAddr returns a localhost address with a randomly assigned free port so
// tests never collide with each other or with a running server.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freeAddr: %v", err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

// minimalDeps builds a RouterDeps with only the fields that setupRouter reads
// at construction time (Cfg). All service/db/vk fields are left nil because
// setupRouter only passes them down to handler constructors — it never calls
// them itself. Tests that need real handler responses should build richer deps.
func minimalDeps(env string) RouterDeps {
	return RouterDeps{
		Cfg: config.Config{
			Env:         env,
			ServiceName: "test-svc",
		},
	}
}

// ---------------------------------------------------------------------------
// setupRouter — route registration
// ---------------------------------------------------------------------------

func TestSetupRouter_HealthRoutes(t *testing.T) {
	e := setupRouter(minimalDeps("development"))
	routes := routeSet(e)

	for _, want := range []string{"GET /health", "GET /health/ready"} {
		if !routes[want] {
			t.Errorf("expected route %q to be registered", want)
		}
	}
}

func TestSetupRouter_DocsRoutes_NonProduction(t *testing.T) {
	for _, env := range []string{"development", "staging", "test", ""} {
		t.Run(env, func(t *testing.T) {
			e := setupRouter(minimalDeps(env))
			routes := routeSet(e)

			for _, want := range []string{"GET /docs", "GET /openapi.yaml"} {
				if !routes[want] {
					t.Errorf("env=%q: expected route %q to be registered", env, want)
				}
			}
		})
	}
}

func TestSetupRouter_DocsRoutes_Production(t *testing.T) {
	e := setupRouter(minimalDeps("production"))
	routes := routeSet(e)

	for _, notWanted := range []string{"GET /docs", "GET /openapi.yaml"} {
		if routes[notWanted] {
			t.Errorf("production: route %q should NOT be registered", notWanted)
		}
	}
}

func TestSetupRouter_HealthRoutes_AlwaysPresent(t *testing.T) {
	// Health routes must exist in ALL envs, including production.
	for _, env := range []string{"production", "staging", "development"} {
		t.Run(env, func(t *testing.T) {
			e := setupRouter(minimalDeps(env))
			routes := routeSet(e)
			for _, want := range []string{"GET /health", "GET /health/ready"} {
				if !routes[want] {
					t.Errorf("env=%q: health route %q missing", env, want)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setupRouter — HTTP responses for health endpoints
// ---------------------------------------------------------------------------

// TestSetupRouter_LiveEndpoint checks that /health returns 200 when the
// handler has no real DB/VK dependency (nil deps — health.NewHandler must
// tolerate nil for the live probe, which only checks the process is up).
//
// If your health.Handler panics on nil deps, replace with a lightweight stub.
func TestSetupRouter_LiveEndpoint_Returns200(t *testing.T) {
	e := setupRouter(minimalDeps("development"))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/health: want 200, got %d", rec.Code)
	}
}

func TestSetupRouter(t *testing.T) {
	// Minimal deps - using nil to not execute the handlers
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

// ---------------------------------------------------------------------------
// startAndAwaitShutdown
// ---------------------------------------------------------------------------

func TestStartAndAwaitShutdown_GracefulShutdown(t *testing.T) {
	e := echo.New()
	e.HideBanner = true
	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	addr := freeAddr(t)
	quit := make(chan os.Signal, 1)

	done := make(chan error, 1)
	go func() {
		done <- startAndAwaitShutdown(e, addr, quit)
	}()

	// Wait until the server is actually accepting connections.
	if err := waitForPort(addr, 2*time.Second); err != nil {
		t.Fatalf("server did not start in time: %v", err)
	}

	// Verify the server handles a real request.
	resp, err := http.Get("http://" + addr + "/ping")
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /ping: want 200, got %d", resp.StatusCode)
	}

	// Send shutdown signal.
	quit <- os.Interrupt

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("startAndAwaitShutdown returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("startAndAwaitShutdown did not return within 5s after signal")
	}
}

func TestStartAndAwaitShutdown_ServerStartError(t *testing.T) {
	// Bind a listener so the port is already taken — Echo will fail to start.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("setup listener: %v", err)
	}
	defer l.Close()
	addr := l.Addr().String()

	e := echo.New()
	e.HideBanner = true

	quit := make(chan os.Signal, 1)

	done := make(chan error, 1)
	go func() {
		done <- startAndAwaitShutdown(e, addr, quit)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error when port is already in use, got nil")
		}
		if !strings.Contains(err.Error(), "server error") {
			t.Errorf("unexpected error message: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("startAndAwaitShutdown did not return within 5s on bind error")
	}
}

func TestStartAndAwaitShutdown_ClosedQuitChannel(t *testing.T) {
	e := echo.New()
	e.HideBanner = true

	addr := freeAddr(t)
	quit := make(chan os.Signal)

	done := make(chan error, 1)
	go func() {
		done <- startAndAwaitShutdown(e, addr, quit)
	}()

	// Wait until the server is up, then close the channel (simulates the
	// program's context being canceled rather than a real OS signal).
	if err := waitForPort(addr, 2*time.Second); err != nil {
		t.Fatalf("server did not start in time: %v", err)
	}
	close(quit)

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("unexpected error on channel close: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not return after channel close")
	}
}

// ---------------------------------------------------------------------------
// waitForPort — poll helper used by shutdown tests
// ---------------------------------------------------------------------------

func waitForPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return &net.OpError{Op: "dial", Net: "tcp", Addr: nil,
		Err: &timeoutErr{addr: addr}}
}

type timeoutErr struct{ addr string }

func (e *timeoutErr) Error() string   { return "timeout waiting for " + e.addr }
func (e *timeoutErr) Timeout() bool   { return true }
func (e *timeoutErr) Temporary() bool { return false }
