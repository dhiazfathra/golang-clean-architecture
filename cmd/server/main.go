package main

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/order"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/config"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

func main() {
	cfg := config.MustLoad()
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
	public := e.Group("")
	protected := e.Group("")
	protected.Use(session.RequireSession(sessionStore))

	auth.RegisterRoutes(public, protected, auth.NewHandler(authSvc))
	rbac.RegisterRoutes(protected.Group("/admin"), rbac.NewHandler(rbacSvc), rbacSvc)
	user.RegisterRoutes(protected, user.NewHandler(userSvc), rbacSvc)
	order.RegisterRoutes(protected, order.NewHandler(orderSvc), rbacSvc)

	e.Logger.Fatal(e.Start(cfg.ListenAddr))
}
