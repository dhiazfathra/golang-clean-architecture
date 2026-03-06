package kvstore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func TestValkeyCache_SetAndGet(t *testing.T) {
	vk := testutil.SetupTestValkey(t)
	c := NewValkeyCache(vk)
	ctx := context.Background()

	err := c.Set(ctx, "kvstore:test_set_get", "1", 10*time.Second)
	require.NoError(t, err)

	val, err := c.Get(ctx, "kvstore:test_set_get")
	require.NoError(t, err)
	assert.Equal(t, "1", val)

	_ = c.Del(ctx, "kvstore:test_set_get")
}

func TestValkeyCache_GetMissing(t *testing.T) {
	vk := testutil.SetupTestValkey(t)
	c := NewValkeyCache(vk)
	ctx := context.Background()

	_, err := c.Get(ctx, "kvstore:nonexistent_key_xyz")
	assert.Error(t, err)
}

func TestValkeyCache_Del(t *testing.T) {
	vk := testutil.SetupTestValkey(t)
	c := NewValkeyCache(vk)
	ctx := context.Background()

	_ = c.Set(ctx, "kvstore:test_del", "1", 10*time.Second)
	err := c.Del(ctx, "kvstore:test_del")
	require.NoError(t, err)

	_, err = c.Get(ctx, "kvstore:test_del")
	assert.Error(t, err)
}

func TestValkeyCache_SetMulti(t *testing.T) {
	vk := testutil.SetupTestValkey(t)
	c := NewValkeyCache(vk)
	ctx := context.Background()

	entries := map[string]string{
		"kvstore:multi_a": "1",
		"kvstore:multi_b": "0",
	}
	err := c.SetMulti(ctx, entries, 10*time.Second)
	require.NoError(t, err)

	val, err := c.Get(ctx, "kvstore:multi_a")
	require.NoError(t, err)
	assert.Equal(t, "1", val)

	val, err = c.Get(ctx, "kvstore:multi_b")
	require.NoError(t, err)
	assert.Equal(t, "0", val)

	_ = c.Del(ctx, "kvstore:multi_a")
	_ = c.Del(ctx, "kvstore:multi_b")
}

func TestValkeyCache_SetMultiEmpty(t *testing.T) {
	vk := testutil.SetupTestValkey(t)
	c := NewValkeyCache(vk)
	ctx := context.Background()

	err := c.SetMulti(ctx, map[string]string{}, 10*time.Second)
	require.NoError(t, err)
}
