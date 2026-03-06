package envvar

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func TestModuleRegistered(t *testing.T) {
	t.Parallel()
	modules := rbac.Modules()
	mod, ok := modules["envvar"]
	assert.True(t, ok, "envvar module should be registered")
	assert.Equal(t, "envvar", mod.Name)
	assert.Contains(t, mod.Fields, "id")
	assert.Contains(t, mod.Fields, "platform")
	assert.Contains(t, mod.Fields, "key")
	assert.Contains(t, mod.Fields, "value")
	assert.Len(t, mod.DefaultPerms, 1)
	assert.Equal(t, "manage", mod.DefaultPerms[0].Action)
}
