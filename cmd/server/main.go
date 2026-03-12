package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/order"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/apitoken"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/config"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/envvar"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/featureflag"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/observability"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

func main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	if err := serve(quit, defaultWire); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

// defaultWire is the real implementation used in production.
func defaultWire(cfg *config.Config) (RouterDeps, func(), error) {
	observability.Init(observability.InitConfig{
		ServiceName:     cfg.ServiceName,
		Env:             cfg.Env,
		StatsdAddr:      cfg.StatsdAddr,
		StatsdNamespace: cfg.StatsdNamespace,
	})

	db := database.MustConnect(cfg.DatabaseURL, database.PoolConfig{
		MaxOpenConns: cfg.DBMaxOpenConns,
		MaxIdleConns: cfg.DBMaxIdleConns,
		ServiceName:  cfg.ServiceName + "-db",
	})

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

	authSvc := auth.NewService(sessionStore, &authUserAdapter{userSvc}, hasher, cfg.SessionTTL)

	orderProjector := order.NewProjector(db)
	orderReadRepo := order.NewPgReadRepository(db)
	orderSvc := order.NewService(es, orderReadRepo, &orderUserProvider{userSvc})

	ctx, cancel := context.WithCancel(context.Background())

	runner := eventstore.NewProjectionRunner(db, es)
	runner.Register(rbacProjector)
	runner.Register(userProjector)
	runner.Register(orderProjector)

	// Seed before starting async projectors to avoid cursor races with RunOnce.
	if err := seeder.Seed(ctx, rbacSvc, &seederUserAdapter{userSvc}, runner,
		cfg.SeedSuperAdminPassword, cfg.SeedDefaultModulePassword); err != nil {
		cancel()
		db.Close()
		vk.Close()
		observability.Stop()
		return RouterDeps{}, nil, fmt.Errorf("seeder: %w", err)
	}

	runner.Start(ctx)

	ffRepo := featureflag.NewRepository(db)
	ffSvc := featureflag.NewService(ffRepo, vk, cfg.FeatureFlagRefreshTTL)
	ffSvc.StartRefresh(ctx)

	evRepo := envvar.NewRepository(db)
	evSvc := envvar.NewService(evRepo, vk, cfg.EnvVarRefreshTTL)
	evSvc.StartRefresh(ctx)

	tokenRepo := apitoken.NewRepository(db)
	tokenSvc := apitoken.NewService(tokenRepo, vk, cfg.APITokenRefreshTTL)
	tokenSvc.StartRefresh(ctx)

	cleanup := func() {
		cancel()
		db.Close()
		vk.Close()
		observability.Stop()
	}

	return RouterDeps{
		Cfg:          *cfg,
		DB:           db,
		VK:           vk,
		SessionStore: sessionStore,
		AuthSvc:      authSvc,
		RBACSvc:      rbacSvc,
		UserSvc:      userSvc,
		OrderSvc:     orderSvc,
		FFSvc:        ffSvc,
		EVSvc:        evSvc,
		TokenSvc:     tokenSvc,
	}, cleanup, nil
}
