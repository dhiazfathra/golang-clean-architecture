package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/valkey-io/valkey-go"

	goapi "github.com/dhiazfathra/golang-clean-architecture/api"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/order"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/config"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/docs"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/featureflag"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/health"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/observability"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

func main() {
	cfg := config.MustLoad()
	observability.Init(observability.InitConfig{
		ServiceName:     cfg.ServiceName,
		Env:             cfg.Env,
		StatsdAddr:      cfg.StatsdAddr,
		StatsdNamespace: cfg.StatsdNamespace,
	})
	defer observability.Stop()

	db := database.MustConnect(cfg.DatabaseURL, database.PoolConfig{
		MaxOpenConns: cfg.DBMaxOpenConns,
		MaxIdleConns: cfg.DBMaxIdleConns,
		ServiceName:  cfg.ServiceName + "-db",
	})
	defer db.Close()

	vk := session.MustConnectValkey(cfg.ValkeyURL)
	defer vk.Close()

	es := eventstore.NewPgStore(db)

	sessionStore := session.NewValkeyStore(vk)
	hasher := auth.NewBcryptHasher()

	rbacProjector := rbac.NewProjector(db)
	rbacRepo := rbac.NewPgReadRepository(db)
	rbacSvc := rbac.NewService(es, rbacRepo)

	userProjector := user.NewProjector(db)
	userReadRepo := user.NewPgReadRepository(db)
	userSvc := user.NewService(es, userReadRepo, hasher)

	authSvc := auth.NewService(sessionStore, &authUserAdapter{userSvc}, hasher, cfg.SessionTTL)

	orderProjector := order.NewProjector(db)
	orderReadRepo := order.NewPgReadRepository(db)
	orderSvc := order.NewService(es, orderReadRepo, &orderUserProvider{userSvc})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := eventstore.NewProjectionRunner(db, es)
	runner.Register(rbacProjector)
	runner.Register(userProjector)
	runner.Register(orderProjector)
	runner.Start(ctx)

	ffRepo := featureflag.NewRepository(db)
	ffSvc := featureflag.NewService(ffRepo, vk, cfg.FeatureFlagRefreshTTL)
	ffSvc.StartRefresh(ctx)

	if err := seeder.Seed(ctx, rbacSvc, &seederUserAdapter{userSvc},
		cfg.SeedSuperAdminPassword, cfg.SeedDefaultModulePassword); err != nil {
		panic("seeder: " + err.Error())
	}

	e := setupRouter(RouterDeps{
		Cfg:          *cfg,
		DB:           db,
		VK:           vk,
		SessionStore: sessionStore,
		AuthSvc:      authSvc,
		RBACSvc:      rbacSvc,
		UserSvc:      userSvc,
		OrderSvc:     orderSvc,
		FFSvc:        ffSvc,
	})

	go func() {
		if err := e.Start(cfg.ListenAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	e.Logger.Info("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}

	e.Logger.Info("server gracefully stopped")
}

// RouterDeps holds all the dependencies required to set up the HTTP router.
type RouterDeps struct {
	Cfg          config.Config
	DB           *sqlx.DB
	VK           valkey.Client
	SessionStore session.SessionStore
	AuthSvc      *auth.Service
	RBACSvc      *rbac.Service
	UserSvc      *user.Service
	OrderSvc     *order.Service
	FFSvc        *featureflag.Service
}

func setupRouter(deps RouterDeps) *echo.Echo {
	e := echo.New()
	e.Use(observability.EchoMiddleware(deps.Cfg.ServiceName))
	e.Use(observability.RequestMetrics())

	public := e.Group("")
	protected := e.Group("")
	protected.Use(session.RequireSession(deps.SessionStore))

	adminGroup := protected.Group("/admin")
	rbacHandler := rbac.NewHandler(deps.RBACSvc, deps.DB)
	userHandler := user.NewHandler(deps.UserSvc, deps.DB)

	auth.RegisterRoutes(public, protected, auth.NewHandler(deps.AuthSvc))
	rbac.RegisterRoutes(adminGroup, rbacHandler, deps.RBACSvc)
	user.RegisterRoutes(protected, userHandler, deps.RBACSvc)
	user.RegisterAdminRoutes(adminGroup, userHandler, deps.RBACSvc)
	order.RegisterRoutes(protected, order.NewHandler(deps.OrderSvc), deps.RBACSvc)

	ffHandler := featureflag.NewHandler(deps.FFSvc)
	featureflag.RegisterAdminRoutes(adminGroup, ffHandler, deps.RBACSvc)

	// Health probes — on root Echo instance, no auth middleware.
	healthHandler := health.NewHandler(deps.DB, deps.VK)
	e.GET("/health", healthHandler.Live)
	e.GET("/health/ready", healthHandler.Ready)

	// API docs — only in non-production environments.
	if deps.Cfg.Env != "production" {
		docsHandler := docs.NewHandler(goapi.Files)
		public.GET("/docs", docsHandler.ScalarUI)
		public.GET("/openapi.yaml", docsHandler.OpenAPISpec)
	}

	return e
}
