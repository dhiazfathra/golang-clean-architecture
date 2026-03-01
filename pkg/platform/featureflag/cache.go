package featureflag

import (
	"context"
	"time"

	"github.com/valkey-io/valkey-go"
)

// cache abstracts the Valkey cache layer so the service is unit-testable
// without a real (or fully-mocked) valkey.Client.
type cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	SetMulti(ctx context.Context, entries map[string]string, ttl time.Duration) error
}

type valkeyCache struct {
	client valkey.Client
}

func newValkeyCache(client valkey.Client) *valkeyCache {
	return &valkeyCache{client: client}
}

func (c *valkeyCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Do(ctx, c.client.B().Get().Key(key).Build()).ToString()
}

func (c *valkeyCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Do(ctx, c.client.B().Set().Key(key).Value(value).Ex(ttl).Build()).Error()
}

func (c *valkeyCache) Del(ctx context.Context, key string) error {
	return c.client.Do(ctx, c.client.B().Del().Key(key).Build()).Error()
}

func (c *valkeyCache) SetMulti(ctx context.Context, entries map[string]string, ttl time.Duration) error {
	cmds := make(valkey.Commands, 0, len(entries))
	for k, v := range entries {
		cmds = append(cmds, c.client.B().Set().Key(k).Value(v).Ex(ttl).Build())
	}
	if len(cmds) > 0 {
		for _, resp := range c.client.DoMulti(ctx, cmds...) {
			if err := resp.Error(); err != nil {
				return err
			}
		}
	}
	return nil
}
