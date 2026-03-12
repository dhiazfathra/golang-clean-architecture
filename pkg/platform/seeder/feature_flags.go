package seeder

import (
	"context"
	"fmt"
)

type FeatureFlagCreator interface {
	Create(ctx context.Context, key, description string, enabled bool, userID string) (*FeatureFlag, error)
	List(ctx context.Context) ([]FeatureFlag, error)
}

type FeatureFlag struct {
	ID          int64
	Key         string
	Enabled     bool
	Description string
}

func SeedFeatureFlags(ctx context.Context, ffSvc FeatureFlagCreator) error {
	existing, err := ffSvc.List(ctx)
	if err != nil {
		return fmt.Errorf("list existing feature flags: %w", err)
	}
	existingKeys := make(map[string]bool)
	for _, f := range existing {
		existingKeys[f.Key] = true
	}

	flags := []struct {
		key         string
		description string
		enabled     bool
	}{
		{"maintenance_mode", "Global maintenance toggle", false},
		{"new_ui", "Feature gate for new UI rollout", false},
		{"rate_limiting", "Enable rate limiting", true},
	}

	for _, f := range flags {
		if existingKeys[f.key] {
			continue
		}
		if _, err := ffSvc.Create(ctx, f.key, f.description, f.enabled, "system"); err != nil {
			return fmt.Errorf("seed feature flag %s: %w", f.key, err)
		}
	}
	return nil
}
