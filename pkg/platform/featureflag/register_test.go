package featureflag

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func TestModuleRegistered(t *testing.T) {
	modules := rbac.Modules()
	mod, ok := modules["featureflag"]
	assert.True(t, ok, "featureflag module should be registered")
	assert.Equal(t, "featureflag", mod.Name)
	assert.Contains(t, mod.Fields, "id")
	assert.Contains(t, mod.Fields, "key")
	assert.Contains(t, mod.Fields, "enabled")
	assert.Len(t, mod.DefaultPerms, 1)
	assert.Equal(t, "manage", mod.DefaultPerms[0].Action)
}
