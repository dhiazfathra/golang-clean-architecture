package seeder

import (
	"context"
	"fmt"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
)

type EnvVarCreator interface {
	Create(ctx context.Context, platform, key, value, userID string) (*EnvVar, error)
	ListByPlatform(ctx context.Context, platform string, req database.PageRequest) (*database.PageResponse[EnvVar], error)
}

type EnvVar struct {
	ID       int64
	Platform string
	Key      string
	Value    string
}

func SeedEnvVars(ctx context.Context, evSvc EnvVarCreator) error {
	platforms := []string{"web", "mobile", "api"}

	for _, platform := range platforms {
		resp, err := evSvc.ListByPlatform(ctx, platform, database.PageRequest{Page: 1, PageSize: 100})
		if err != nil {
			return fmt.Errorf("list env vars for %s: %w", platform, err)
		}

		existingKeys := make(map[string]bool)
		for _, ev := range resp.Items {
			existingKeys[ev.Key] = true
		}

		vars := []struct {
			key   string
			value string
		}{
			{"APP_NAME", "Go Clean Architecture"},
			{"LOG_LEVEL", "info"},
			{"TIMEOUT_SECONDS", "30"},
		}

		for _, v := range vars {
			if existingKeys[v.key] {
				continue
			}
			if _, err := evSvc.Create(ctx, platform, v.key, v.value, "system"); err != nil {
				return fmt.Errorf("seed env var %s/%s: %w", platform, v.key, err)
			}
		}
	}
	return nil
}
