package kvstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Get_L1_Hit(t *testing.T) {
	mc := newMockCache()
	s := NewStore(mc, "test:", 30*time.Second, nil)
	s.local.Store("mykey", "myval")

	val, ok := s.Get("mykey", nil)
	assert.True(t, ok)
	assert.Equal(t, "myval", val)
}

func TestStore_Get_L2_Hit(t *testing.T) {
	mc := newMockCache()
	mc.data["test:mykey"] = "cached_val"
	s := NewStore(mc, "test:", 30*time.Second, nil)

	val, ok := s.Get("mykey", nil)
	assert.True(t, ok)
	assert.Equal(t, "cached_val", val)

	// Verify promoted to L1
	v, ok := s.Local("mykey")
	assert.True(t, ok)
	assert.Equal(t, "cached_val", v)
}

func TestStore_Get_L3_Fallback(t *testing.T) {
	mc := newMockCache()
	s := NewStore(mc, "test:", 30*time.Second, nil)

	fallback := func() (string, error) {
		return "db_val", nil
	}

	val, ok := s.Get("mykey", fallback)
	assert.True(t, ok)
	assert.Equal(t, "db_val", val)

	// Verify backfill to L1
	v, ok := s.Local("mykey")
	assert.True(t, ok)
	assert.Equal(t, "db_val", v)

	// Verify backfill to L2
	cached, err := mc.Get(context.Background(), "test:mykey")
	assert.NoError(t, err)
	assert.Equal(t, "db_val", cached)
}

func TestStore_Get_AllMiss(t *testing.T) {
	mc := newMockCache()
	s := NewStore(mc, "test:", 30*time.Second, nil)

	val, ok := s.Get("missing", nil)
	assert.False(t, ok)
	assert.Equal(t, "", val)
}

func TestStore_Get_FallbackError(t *testing.T) {
	mc := newMockCache()
	s := NewStore(mc, "test:", 30*time.Second, nil)

	fallback := func() (string, error) {
		return "", errors.New("db error")
	}

	val, ok := s.Get("missing", fallback)
	assert.False(t, ok)
	assert.Equal(t, "", val)
}

func TestStore_Set(t *testing.T) {
	mc := newMockCache()
	s := NewStore(mc, "test:", 30*time.Second, nil)

	s.Set(context.Background(), "mykey", "myval")

	// L1
	v, ok := s.Local("mykey")
	assert.True(t, ok)
	assert.Equal(t, "myval", v)

	// L2
	cached, err := mc.Get(context.Background(), "test:mykey")
	assert.NoError(t, err)
	assert.Equal(t, "myval", cached)
}

func TestStore_Delete(t *testing.T) {
	mc := newMockCache()
	s := NewStore(mc, "test:", 30*time.Second, nil)

	s.Set(context.Background(), "mykey", "myval")
	s.Delete(context.Background(), "mykey")

	_, ok := s.Local("mykey")
	assert.False(t, ok)

	_, err := mc.Get(context.Background(), "test:mykey")
	assert.Error(t, err)
}

func TestStore_Reload_PopulatesCache(t *testing.T) {
	mc := newMockCache()
	loader := func(_ context.Context) (map[string]string, error) {
		return map[string]string{"a": "1", "b": "2"}, nil
	}
	s := NewStore(mc, "test:", 30*time.Second, loader)

	err := s.reload(context.Background())
	require.NoError(t, err)

	v, ok := s.Local("a")
	assert.True(t, ok)
	assert.Equal(t, "1", v)

	v, ok = s.Local("b")
	assert.True(t, ok)
	assert.Equal(t, "2", v)
}

func TestStore_Reload_PrunesStaleKeys(t *testing.T) {
	mc := newMockCache()
	loader := func(_ context.Context) (map[string]string, error) {
		return map[string]string{"fresh": "val"}, nil
	}
	s := NewStore(mc, "test:", 30*time.Second, loader)
	s.local.Store("stale", "old")

	err := s.reload(context.Background())
	require.NoError(t, err)

	_, ok := s.Local("stale")
	assert.False(t, ok, "stale key should be pruned")

	v, ok := s.Local("fresh")
	assert.True(t, ok)
	assert.Equal(t, "val", v)
}

func TestStore_Reload_DBError(t *testing.T) {
	mc := newMockCache()
	loader := func(_ context.Context) (map[string]string, error) {
		return nil, errors.New("db error")
	}
	s := NewStore(mc, "test:", 30*time.Second, loader)

	err := s.reload(context.Background())
	require.Error(t, err)
}

func TestStore_Reload_Empty(t *testing.T) {
	mc := newMockCache()
	loader := func(_ context.Context) (map[string]string, error) {
		return map[string]string{}, nil
	}
	s := NewStore(mc, "test:", 30*time.Second, loader)

	err := s.reload(context.Background())
	require.NoError(t, err)
}

func TestStore_StartRefresh_RunsAndCancels(t *testing.T) {
	mc := newMockCache()
	called := 0
	loader := func(_ context.Context) (map[string]string, error) {
		called++
		return map[string]string{}, nil
	}
	s := NewStore(mc, "test:", 50*time.Millisecond, loader)

	ctx, cancel := context.WithCancel(context.Background())
	s.StartRefresh(ctx)
	time.Sleep(80 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
	assert.GreaterOrEqual(t, called, 1)
}

func TestStore_Local_Miss(t *testing.T) {
	mc := newMockCache()
	s := NewStore(mc, "test:", 30*time.Second, nil)

	_, ok := s.Local("nonexistent")
	assert.False(t, ok)
}
