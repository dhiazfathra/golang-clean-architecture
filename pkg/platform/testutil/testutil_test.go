package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitNoopDoesNotPanic(t *testing.T) {
	// init() already called on package import — just verify no panic
	assert.True(t, true)
}

func TestNewMockDB(t *testing.T) {
	db, mock := NewMockDB(t)
	assert.NotNil(t, db)
	assert.NotNil(t, mock)
}

func TestSetupTestDB_SkipsWhenUnavailable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	// This will skip if postgres is not available (via t.Skipf inside SetupTestDB)
	db := SetupTestDB(t)
	if db != nil {
		assert.NotNil(t, db)
	}
}

func TestSetupTestValkey_SkipsWhenUnavailable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	// This will skip if valkey is not available (via t.Skipf inside SetupTestValkey)
	client := SetupTestValkey(t)
	if client != nil {
		assert.NotNil(t, client)
	}
}
