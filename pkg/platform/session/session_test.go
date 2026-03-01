package session_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func TestNewValkeyStore(t *testing.T) {
	client := testutil.SetupTestValkey(t)
	store := session.NewValkeyStore(client)
	assert.NotNil(t, store)
}

func TestMustConnectValkeyPanic(t *testing.T) {
	assert.Panics(t, func() {
		session.MustConnectValkey("invalid-host:99999")
	})
}
