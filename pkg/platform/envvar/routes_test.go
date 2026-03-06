package envvar

import (
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func TestRegisterAdminRoutes(t *testing.T) {
	t.Parallel()
	db, _ := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := newMockCache()
	svc := newServiceWithCache(repo, mc, 30*time.Second)
	h := NewHandler(svc)

	e := echo.New()
	g := e.Group("/admin")

	// rbacSvc is nil — we only check that routes are registered without panic.
	// RBAC middleware is tested separately.
	RegisterAdminRoutes(g, h, nil)

	routes := e.Routes()
	assert.True(t, len(routes) > 0, "expected routes to be registered")
}
