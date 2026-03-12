package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/valkey-io/valkey-go"

	goapi "github.com/dhiazfathra/golang-clean-architecture/api"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/order"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/apitoken"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/config"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/docs"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/envvar"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/featureflag"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/health"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/observability"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

type wireFn func(cfg *config.Config) (RouterDeps, func(), error)

// serve wires all dependencies, starts the HTTP server, and blocks until a
// signal arrives on quit (or the channel is closed). Accepting the signal
// channel as a parameter makes the entire shutdown path testable without
// spawning a real OS process.
func serve(quit <-chan os.Signal, wire wireFn) error {
	cfg := config.MustLoad()
	deps, cleanup, err := wire(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	e := setupRouter(deps)
	return startAndAwaitShutdown(e, cfg.ListenAddr, quit, deps.Logger)
}

// startAndAwaitShutdown starts e in a goroutine, waits for a quit signal or a
// hard server error, then performs a 30-second graceful shutdown.
// It is its own function so tests can drive it with a synthetic signal channel
// and a pre-configured Echo instance — no real infrastructure required.
func startAndAwaitShutdown(e *echo.Echo, addr string, quit <-chan os.Signal, logger zerolog.Logger) error {
	serverErr := make(chan error, 1)

	go func() {
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-quit:
		// normal shutdown path — fall through
	}

	logger.Info().Msg("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	logger.Info().Msg("server gracefully stopped")
	return nil
}

// ---------------------------------------------------------------------------
// RouterDeps + setupRouter
// ---------------------------------------------------------------------------

// RouterDeps holds all the dependencies required to set up the HTTP router.
type RouterDeps struct {
	Cfg          config.Config
	DB           *sqlx.DB
	VK           valkey.Client
	Logger       zerolog.Logger
	SessionStore session.SessionStore
	AuthSvc      *auth.Service
	RBACSvc      *rbac.Service
	UserSvc      *user.Service
	OrderSvc     *order.Service
	FFSvc        *featureflag.Service
	EVSvc        *envvar.Service
	TokenSvc     *apitoken.Service
}

func setupRouter(deps RouterDeps) *echo.Echo {
	e := echo.New()
	e.Use(observability.EchoMiddleware(deps.Cfg.ServiceName))
	e.Use(observability.RequestMetrics())

	v1 := e.Group("/api/v1")
	public := v1.Group("")
	protected := v1.Group("")
	protected.Use(session.RequireSession(deps.SessionStore))

	l := deps.Logger
	adminGroup := protected.Group("/admin")
	rbacHandler := rbac.NewHandler(deps.RBACSvc, deps.DB, l)
	userHandler := user.NewHandler(deps.UserSvc, deps.DB, l)

	auth.RegisterRoutes(public, protected, auth.NewHandler(deps.AuthSvc))
	rbac.RegisterRoutes(adminGroup, rbacHandler, deps.RBACSvc, l)
	user.RegisterRoutes(protected, userHandler, deps.RBACSvc, l)
	user.RegisterAdminRoutes(adminGroup, userHandler, deps.RBACSvc, l)
	order.RegisterRoutes(protected, order.NewHandler(deps.OrderSvc, l), deps.RBACSvc, l)

	// Multi-auth group: accepts both session cookies and Bearer tokens.
	multiAuth := v1.Group("")
	multiAuth.Use(session.RequireMultiAuth(deps.SessionStore, deps.TokenSvc))
	multiAuthAdmin := multiAuth.Group("/admin")

	ffHandler := featureflag.NewHandler(deps.FFSvc, l)
	featureflag.RegisterAdminRoutes(multiAuthAdmin, ffHandler, deps.RBACSvc, l)

	evHandler := envvar.NewHandler(deps.EVSvc, l)
	envvar.RegisterRoutes(multiAuth, evHandler, deps.RBACSvc, l)

	// API token management — session-only (under original adminGroup).
	tokenHandler := apitoken.NewHandler(deps.TokenSvc, l)
	apitoken.RegisterRoutes(adminGroup, tokenHandler, deps.RBACSvc, l)

	// Health probes — on root Echo instance, no auth middleware.
	healthHandler := health.NewHandler(deps.DB, deps.VK)
	e.GET("/health", healthHandler.Live)
	e.GET("/health/ready", healthHandler.Ready)

	// API docs — only in non-production environments.
	if deps.Cfg.Env != "production" {
		docsHandler := docs.NewHandler(goapi.Files)
		e.GET("/docs", docsHandler.ScalarUI)
		e.GET("/openapi.yaml", docsHandler.OpenAPISpec)
	}

	return e
}
