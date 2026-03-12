package seeder

import (
	"context"
	"fmt"
	"time"
)

type APITokenCreator interface {
	Create(ctx context.Context, name, userID string, ttl time.Duration) (string, *APIToken, error)
	List(ctx context.Context, userID string) ([]APIToken, error)
}

type APIToken struct {
	ID          int64
	Name        string
	TokenHash   string
	TokenPrefix string
	UserID      string
	ExpiresAt   time.Time
}

func SeedAPITokens(ctx context.Context, tokenSvc APITokenCreator, superAdminUserID string) error {
	existing, err := tokenSvc.List(ctx, superAdminUserID)
	if err != nil {
		return fmt.Errorf("list existing api tokens: %w", err)
	}

	existingNames := make(map[string]bool)
	for _, t := range existing {
		existingNames[t.Name] = true
	}

	if !existingNames["dev_token"] {
		raw, _, err := tokenSvc.Create(ctx, "dev_token", superAdminUserID, 365*24*time.Hour)
		if err != nil {
			return fmt.Errorf("seed dev api token: %w", err)
		}
		fmt.Printf("Created dev API token: %s\n", raw) //nolint:forbidigo // TODO: replace with proper logging
	}

	return nil
}
