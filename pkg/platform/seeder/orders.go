package seeder

import (
	"context"
	"fmt"
)

type OrderCreator interface {
	CreateOrder(ctx context.Context, cmd CreateOrderCmd) (string, error)
	List(ctx context.Context, req ListRequest) (*ListResponse, error)
}

type CreateOrderCmd struct {
	UserID string
	Total  float64
	Actor  string
}

type ListRequest struct {
	Page     int
	PageSize int
}

type ListResponse struct {
	Items []OrderReadModel
	Total int64
}

type OrderReadModel struct {
	ID     string
	UserID string
	Total  float64
}

func SeedOrders(ctx context.Context, orderSvc OrderCreator, userIDs []string) error {
	existing, err := orderSvc.List(ctx, ListRequest{Page: 1, PageSize: 100})
	if err != nil {
		return fmt.Errorf("list existing orders: %w", err)
	}

	if len(existing.Items) > 0 {
		return nil
	}

	if len(userIDs) == 0 {
		return nil
	}

	orders := []struct {
		userID string
		total  float64
	}{
		{userIDs[0], 99.99},
		{userIDs[0], 149.50},
	}

	if len(userIDs) > 1 {
		orders = append(orders, struct {
			userID string
			total  float64
		}{userIDs[1], 299.00})
	}

	for _, o := range orders {
		if _, err := orderSvc.CreateOrder(ctx, CreateOrderCmd{
			UserID: o.userID,
			Total:  o.total,
			Actor:  "system",
		}); err != nil {
			return fmt.Errorf("seed order for user %s: %w", o.userID, err)
		}
	}

	return nil
}
