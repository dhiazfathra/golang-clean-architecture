package main

import (
	"context"

	"github.com/labstack/echo/v4"

	goapi "github.com/dhiazfathra/golang-clean-architecture/api"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/order"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/config"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/docs"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/health"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/observability"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

func main() {
	cfg := config.MustLoad()
	observability.Init("golang-clean-arch", cfg.Env)
	defer observability.Stop()

	db := database.MustConnect(cfg.DatabaseURL)
	vk := session.MustConnectValkey(cfg.ValkeyURL)
	es := eventstore.NewPgStore(db)

	sessionStore := session.NewValkeyStore(vk)
	hasher := auth.NewBcryptHasher()

	rbacProjector := rbac.NewProjector(db)
	rbacRepo := rbac.NewPgReadRepository(db)
	rbacSvc := rbac.NewService(es, rbacRepo)

	userProjector := user.NewProjector(db)
	userReadRepo := user.NewPgReadRepository(db)
	userSvc := user.NewService(es, userReadRepo, hasher)

	authSvc := auth.NewService(sessionStore, &authUserAdapter{userSvc}, hasher)

	orderProjector := order.NewProjector(db)
	orderReadRepo := order.NewPgReadRepository(db)
	orderSvc := order.NewService(es, orderReadRepo, &orderUserProvider{userSvc})

	runner := eventstore.NewProjectionRunner(db, es)
	runner.Register(rbacProjector)
	runner.Register(userProjector)
	runner.Register(orderProjector)
	runner.Start(context.Background())

	if err := seeder.Seed(context.Background(), rbacSvc, &seederUserAdapter{userSvc},
		cfg.SeedSuperAdminPassword, cfg.SeedDefaultModulePassword); err != nil {
		panic("seeder: " + err.Error())
	}

	e := echo.New()
	e.Use(observability.EchoMiddleware("golang-clean-arch"))
	e.Use(observability.RequestMetrics())
	public := e.Group("")
	protected := e.Group("")
	protected.Use(session.RequireSession(sessionStore))

	adminGroup := protected.Group("/admin")
	rbacHandler := rbac.NewHandler(rbacSvc, db)
	userHandler := user.NewHandler(userSvc, db)

	auth.RegisterRoutes(public, protected, auth.NewHandler(authSvc))
	rbac.RegisterRoutes(adminGroup, rbacHandler, rbacSvc)
	user.RegisterRoutes(protected, userHandler, rbacSvc)
	user.RegisterAdminRoutes(adminGroup, userHandler, rbacSvc)
	order.RegisterRoutes(protected, order.NewHandler(orderSvc), rbacSvc)

	// Health probes — on root Echo instance, no auth middleware.
	healthHandler := health.NewHandler(db, vk)
	e.GET("/health", healthHandler.Live)
	e.GET("/health/ready", healthHandler.Ready)

	// API docs — only in non-production environments.
	if cfg.Env != "production" {
		docsHandler := docs.NewHandler(goapi.Files)
		public.GET("/docs", docsHandler.ScalarUI)
		public.GET("/openapi.yaml", docsHandler.OpenAPISpec)
	}

	e.Logger.Fatal(e.Start(cfg.ListenAddr))
}
