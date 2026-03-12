package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/order"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/apitoken"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/config"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/envvar"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/featureflag"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/logging"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
	fmt.Println("✓ Seeding completed successfully") //nolint:forbidigo
}

func run() error {
	cfg := config.MustLoad()
	logger := logging.New(cfg.Env, cfg.LogLevel)

	db := database.MustConnect(cfg.DatabaseURL, database.PoolConfig{
		MaxOpenConns: cfg.DBMaxOpenConns,
		MaxIdleConns: cfg.DBMaxIdleConns,
		ServiceName:  cfg.ServiceName + "-seed",
	})
	defer db.Close()

	vk := session.MustConnectValkey(cfg.ValkeyURL)
	defer vk.Close()

	es := eventstore.NewPgStore(db)
	hasher := auth.NewBcryptHasher()

	rbacProjector := rbac.NewProjector(db)
	rbacRepo := rbac.NewPgReadRepository(db)
	rbacSvc := rbac.NewService(es, rbacRepo)

	userProjector := user.NewProjector(db)
	userReadRepo := user.NewPgReadRepository(db)
	userSvc := user.NewService(es, userReadRepo, hasher)

	orderProjector := order.NewProjector(db)
	orderReadRepo := order.NewPgReadRepository(db)
	orderSvc := order.NewService(es, orderReadRepo, &orderUserProvider{userSvc})

	ctx := context.Background()

	runner := eventstore.NewProjectionRunner(db, es, logger)
	runner.Register(rbacProjector)
	runner.Register(userProjector)
	runner.Register(orderProjector)

	ffRepo := featureflag.NewRepository(db)
	ffSvc := featureflag.NewService(ffRepo, vk, cfg.FeatureFlagRefreshTTL)

	evRepo := envvar.NewRepository(db)
	evSvc := envvar.NewService(evRepo, vk, cfg.EnvVarRefreshTTL)

	tokenRepo := apitoken.NewRepository(db)
	tokenSvc := apitoken.NewService(tokenRepo, vk, cfg.APITokenRefreshTTL)

	return seeder.Seed(ctx, seeder.SeedParams{
		RBACService:           rbacSvc,
		UserService:           &seederUserAdapter{userSvc},
		FeatureFlagService:    &seederFFAdapter{ffSvc},
		EnvVarService:         &seederEnvVarAdapter{evSvc},
		APITokenService:       &seederAPITokenAdapter{tokenSvc},
		OrderService:          &seederOrderAdapter{orderSvc},
		Flusher:               runner,
		Logger:                logger,
		SuperAdminPassword:    cfg.SeedSuperAdminPassword,
		DefaultModulePassword: cfg.SeedDefaultModulePassword,
	})
}
