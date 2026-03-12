package main

import (
	"context"
	"time"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/order"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/apitoken"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/envvar"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/featureflag"
)

// ffService is the subset of featureflag.Service used by seederFFAdapter.
type ffService interface {
	Create(ctx context.Context, key, description string, enabled bool, userID string) (*featureflag.Flag, error)
	List(ctx context.Context) ([]featureflag.Flag, error)
}

// evService is the subset of envvar.Service used by seederEnvVarAdapter.
type evService interface {
	Create(ctx context.Context, platform, key, value, userID string) (*envvar.EnvVar, error)
	ListByPlatform(ctx context.Context, platform string, req database.PageRequest) (*database.PageResponse[envvar.EnvVar], error)
}

// atService is the subset of apitoken.Service used by seederAPITokenAdapter.
type atService interface {
	Create(ctx context.Context, name, userID string, ttl time.Duration) (string, *apitoken.APIToken, error)
	List(ctx context.Context, userID string) ([]apitoken.APIToken, error)
}

// orderService is the subset of order.Service used by seederOrderAdapter.
type orderService interface {
	CreateOrder(ctx context.Context, cmd order.CreateOrderCmd) (string, error)
	List(ctx context.Context, req order.ListRequest) (*order.ListResponse, error)
}
