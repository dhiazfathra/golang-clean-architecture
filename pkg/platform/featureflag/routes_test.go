package featureflag

import (
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func TestRegisterAdminRoutes(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := newMockCache()
	svc := newServiceWithCache(repo, mc, 30*time.Second)
	h := NewHandler(svc)

	e := echo.New()
	g := e.Group("/admin")
	RegisterAdminRoutes(g, h, nil)

	routes := e.Routes()
	paths := make(map[string]bool)
	for _, r := range routes {
		paths[r.Method+":"+r.Path] = true
	}

	assert.True(t, paths["GET:/admin/feature-flags"])
	assert.True(t, paths["POST:/admin/feature-flags"])
	assert.True(t, paths["PATCH:/admin/feature-flags/:key"])
	assert.True(t, paths["DELETE:/admin/feature-flags/:key"])
}
