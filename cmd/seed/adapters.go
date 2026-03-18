package main

import (
	"context"
	"strconv"
	"time"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/order"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/apitoken"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/envvar"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/featureflag"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
)

type ffService interface {
	Create(ctx context.Context, key, description string, enabled bool, userID string) (*featureflag.Flag, error)
	List(ctx context.Context) ([]featureflag.Flag, error)
}

type evService interface {
	Create(ctx context.Context, platform, key, value, userID string) (*envvar.EnvVar, error)
	ListByPlatform(ctx context.Context, platform string, req database.PageRequest) (*database.PageResponse[envvar.EnvVar], error)
}

type atService interface {
	Create(ctx context.Context, name, userID string, ttl time.Duration) (string, *apitoken.APIToken, error)
	List(ctx context.Context, userID string) ([]apitoken.APIToken, error)
}

type orderSvc interface {
	CreateOrder(ctx context.Context, cmd order.CreateOrderCmd) (string, error)
	List(ctx context.Context, req order.ListRequest) (*order.ListResponse, error)
}

type orderUserProvider struct{ svc user.UserService }

func (a *orderUserProvider) GetByID(ctx context.Context, id string) (bool, error) {
	u, err := a.svc.GetByID(ctx, id)
	return u != nil, err
}

type seederUserAdapter struct{ svc user.UserService }

func (a *seederUserAdapter) CreateUser(ctx context.Context, cmd seeder.CreateUserCmd) (string, error) {
	return a.svc.CreateUserForSeeder(ctx, cmd)
}

func (a *seederUserAdapter) GetByEmail(ctx context.Context, email string) (*seeder.UserRecord, error) {
	return a.svc.GetByEmailForSeeder(ctx, email)
}

func (a *seederUserAdapter) AssignRole(ctx context.Context, userID, roleID, actor string) error {
	return a.svc.AssignRole(ctx, userID, roleID, actor)
}

type seederFFAdapter struct{ svc ffService }

func (a *seederFFAdapter) Create(ctx context.Context, key, description string, enabled bool, userID string) (*seeder.FeatureFlag, error) {
	f, err := a.svc.Create(ctx, key, description, enabled, userID)
	if err != nil {
		return nil, err
	}
	return &seeder.FeatureFlag{
		ID:          f.ID,
		Key:         f.Key,
		Enabled:     f.Enabled,
		Description: f.Description,
	}, nil
}

func (a *seederFFAdapter) List(ctx context.Context) ([]seeder.FeatureFlag, error) {
	flags, err := a.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]seeder.FeatureFlag, len(flags))
	for i, f := range flags {
		result[i] = seeder.FeatureFlag{
			ID:          f.ID,
			Key:         f.Key,
			Enabled:     f.Enabled,
			Description: f.Description,
		}
	}
	return result, nil
}

type seederEnvVarAdapter struct{ svc evService }

func (a *seederEnvVarAdapter) Create(ctx context.Context, platform, key, value, userID string) (*seeder.EnvVar, error) {
	e, err := a.svc.Create(ctx, platform, key, value, userID)
	if err != nil {
		return nil, err
	}
	return &seeder.EnvVar{
		ID:       e.ID,
		Platform: e.Platform,
		Key:      e.Key,
		Value:    e.Value,
	}, nil
}

func (a *seederEnvVarAdapter) ListByPlatform(ctx context.Context, platform string, req database.PageRequest) (*database.PageResponse[seeder.EnvVar], error) {
	resp, err := a.svc.ListByPlatform(ctx, platform, req)
	if err != nil {
		return nil, err
	}
	items := make([]seeder.EnvVar, len(resp.Items))
	for i, e := range resp.Items {
		items[i] = seeder.EnvVar{
			ID:       e.ID,
			Platform: e.Platform,
			Key:      e.Key,
			Value:    e.Value,
		}
	}
	return &database.PageResponse[seeder.EnvVar]{
		Items:      items,
		Total:      resp.Total,
		Page:       resp.Page,
		PageSize:   resp.PageSize,
		TotalPages: resp.TotalPages,
	}, nil
}

type seederAPITokenAdapter struct{ svc atService }

func (a *seederAPITokenAdapter) Create(ctx context.Context, name, userID string, ttl time.Duration) (string, *seeder.APIToken, error) {
	raw, token, err := a.svc.Create(ctx, name, userID, ttl)
	if err != nil {
		return "", nil, err
	}
	return raw, &seeder.APIToken{
		ID:          token.ID,
		Name:        token.Name,
		TokenHash:   token.TokenHash,
		TokenPrefix: token.TokenPrefix,
		UserID:      token.UserID,
		ExpiresAt:   token.ExpiresAt,
	}, nil
}

func (a *seederAPITokenAdapter) List(ctx context.Context, userID string) ([]seeder.APIToken, error) {
	tokens, err := a.svc.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]seeder.APIToken, len(tokens))
	for i, t := range tokens {
		result[i] = seeder.APIToken{
			ID:          t.ID,
			Name:        t.Name,
			TokenHash:   t.TokenHash,
			TokenPrefix: t.TokenPrefix,
			UserID:      t.UserID,
			ExpiresAt:   t.ExpiresAt,
		}
	}
	return result, nil
}

type seederOrderAdapter struct{ svc orderSvc }

func (a *seederOrderAdapter) CreateOrder(ctx context.Context, cmd seeder.CreateOrderCmd) (string, error) {
	return a.svc.CreateOrder(ctx, order.CreateOrderCmd{
		UserID: cmd.UserID,
		Total:  cmd.Total,
		Actor:  cmd.Actor,
	})
}

func (a *seederOrderAdapter) List(ctx context.Context, req seeder.ListRequest) (*seeder.ListResponse, error) {
	resp, err := a.svc.List(ctx, order.ListRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	items := make([]seeder.OrderReadModel, len(resp.Items))
	for i, o := range resp.Items {
		items[i] = seeder.OrderReadModel{
			ID:     strconv.FormatInt(o.ID, 10),
			UserID: strconv.FormatInt(o.UserID, 10),
			Total:  o.Total,
		}
	}
	return &seeder.ListResponse{
		Items: items,
		Total: resp.Total,
	}, nil
}
