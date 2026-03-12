package envvar

import (
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/logging"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()
	db, _ := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := newMockCache()
	svc := newServiceWithStore(repo, mc, 30*time.Second)
	h := NewHandler(svc, logging.Noop())

	e := echo.New()
	g := e.Group("")

	// rbacSvc is nil — we only check that routes are registered without panic.
	// RBAC middleware is tested separately.
	RegisterRoutes(g, h, nil, logging.Noop())

	routes := e.Routes()
	assert.True(t, len(routes) > 0, "expected routes to be registered")
}
