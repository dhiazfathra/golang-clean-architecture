package envvar

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func TestValkeyCache_SetGetDel(t *testing.T) {
	t.Parallel()
	vk := testutil.SetupTestValkey(t)
	c := newValkeyCache(vk)
	ctx := context.Background()

	err := c.Set(ctx, "envvar_test:k1", "v1", 10*time.Second)
	require.NoError(t, err)

	val, err := c.Get(ctx, "envvar_test:k1")
	require.NoError(t, err)
	assert.Equal(t, "v1", val)

	err = c.Del(ctx, "envvar_test:k1")
	require.NoError(t, err)

	_, err = c.Get(ctx, "envvar_test:k1")
	assert.Error(t, err)
}

func TestValkeyCache_SetMulti(t *testing.T) {
	t.Parallel()
	vk := testutil.SetupTestValkey(t)
	c := newValkeyCache(vk)
	ctx := context.Background()

	entries := map[string]string{
		"envvar_test:m1": "val1",
		"envvar_test:m2": "val2",
	}
	err := c.SetMulti(ctx, entries, 10*time.Second)
	require.NoError(t, err)

	v1, err := c.Get(ctx, "envvar_test:m1")
	require.NoError(t, err)
	assert.Equal(t, "val1", v1)

	v2, err := c.Get(ctx, "envvar_test:m2")
	require.NoError(t, err)
	assert.Equal(t, "val2", v2)

	// Cleanup
	_ = c.Del(ctx, "envvar_test:m1")
	_ = c.Del(ctx, "envvar_test:m2")
}

func TestValkeyCache_SetMulti_Empty(t *testing.T) {
	t.Parallel()
	vk := testutil.SetupTestValkey(t)
	c := newValkeyCache(vk)
	ctx := context.Background()

	err := c.SetMulti(ctx, map[string]string{}, 10*time.Second)
	require.NoError(t, err)
}
