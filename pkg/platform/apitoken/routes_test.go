package apitoken

import (
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/kvstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func TestRegisterRoutes(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := kvstore.NewMockCache()
	svc := newServiceWithStore(repo, mc, 30*time.Second)
	h := NewHandler(svc)

	e := echo.New()
	g := e.Group("/admin")
	RegisterRoutes(g, h, nil)

	routes := e.Routes()
	paths := make(map[string]bool)
	for _, r := range routes {
		paths[r.Method+":"+r.Path] = true
	}

	assert.True(t, paths["POST:/admin/api-tokens"])
	assert.True(t, paths["GET:/admin/api-tokens"])
	assert.True(t, paths["DELETE:/admin/api-tokens/:id"])
}
