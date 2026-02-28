package rbac

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
)

// --- hand-rolled mocks ---

type mockEventStore struct {
	AppendFn func(ctx context.Context, events []eventstore.Event) error
	LoadFn   func(ctx context.Context, aggType, aggID string, fromVersion int) ([]eventstore.Event, error)
}

func (m *mockEventStore) Append(ctx context.Context, events []eventstore.Event) error {
	if m.AppendFn != nil {
		return m.AppendFn(ctx, events)
	}
	return nil
}

func (m *mockEventStore) Load(ctx context.Context, aggType, aggID string, fromVersion int) ([]eventstore.Event, error) {
	if m.LoadFn != nil {
		return m.LoadFn(ctx, aggType, aggID, fromVersion)
	}
	return nil, nil
}

type mockRepo struct {
	GetRoleByIDFn          func(ctx context.Context, id string) (*RoleReadModel, error)
	GetRoleByNameFn        func(ctx context.Context, name string) (*RoleReadModel, error)
	ListRolesFn            func(ctx context.Context) ([]RoleReadModel, error)
	GetPermissionsForRoleFn func(ctx context.Context, roleID string) ([]PermissionReadModel, error)
	GetRolesForUserFn      func(ctx context.Context, userID string) ([]string, error)
}

func (m *mockRepo) GetRoleByID(ctx context.Context, id string) (*RoleReadModel, error) {
	return m.GetRoleByIDFn(ctx, id)
}
func (m *mockRepo) GetRoleByName(ctx context.Context, name string) (*RoleReadModel, error) {
	return m.GetRoleByNameFn(ctx, name)
}
func (m *mockRepo) ListRoles(ctx context.Context) ([]RoleReadModel, error) {
	return m.ListRolesFn(ctx)
}
func (m *mockRepo) GetPermissionsForRole(ctx context.Context, roleID string) ([]PermissionReadModel, error) {
	return m.GetPermissionsForRoleFn(ctx, roleID)
}
func (m *mockRepo) GetRolesForUser(ctx context.Context, userID string) ([]string, error) {
	return m.GetRolesForUserFn(ctx, userID)
}

// --- CheckPermission tests ---

func TestCheckPermission_WildcardRole(t *testing.T) {
	repo := &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"role_superadmin"}, nil
		},
		GetPermissionsForRoleFn: func(_ context.Context, _ string) ([]PermissionReadModel, error) {
			return []PermissionReadModel{
				{Module: "*", Action: "*", FieldMode: "all"},
			}, nil
		},
	}
	svc := NewService(&mockEventStore{}, repo)

	allowed, policy, err := svc.CheckPermission(context.Background(), "user1", "orders", "delete")
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, "all", policy.Mode)
}

func TestCheckPermission_NoRole(t *testing.T) {
	repo := &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, nil
		},
	}
	svc := NewService(&mockEventStore{}, repo)

	allowed, _, err := svc.CheckPermission(context.Background(), "user1", "orders", "read")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestCheckPermission_ExactMatch(t *testing.T) {
	repo := &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"role_viewer"}, nil
		},
		GetPermissionsForRoleFn: func(_ context.Context, _ string) ([]PermissionReadModel, error) {
			return []PermissionReadModel{
				{Module: "orders", Action: "read", FieldMode: "allow", FieldList: []string{"id", "status"}},
			}, nil
		},
	}
	svc := NewService(&mockEventStore{}, repo)

	allowed, policy, err := svc.CheckPermission(context.Background(), "user1", "orders", "read")
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, "allow", policy.Mode)
	assert.ElementsMatch(t, []string{"id", "status"}, policy.Fields)
}

func TestCheckPermission_NoMatchingAction(t *testing.T) {
	repo := &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"role_viewer"}, nil
		},
		GetPermissionsForRoleFn: func(_ context.Context, _ string) ([]PermissionReadModel, error) {
			return []PermissionReadModel{
				{Module: "orders", Action: "read", FieldMode: "all"},
			}, nil
		},
	}
	svc := NewService(&mockEventStore{}, repo)

	allowed, _, err := svc.CheckPermission(context.Background(), "user1", "orders", "delete")
	require.NoError(t, err)
	assert.False(t, allowed)
}

// --- mergeFieldPolicies tests ---

func TestMergeFieldPolicies_AllWins(t *testing.T) {
	policies := []FieldPolicy{
		{Mode: "allow", Fields: []string{"id"}},
		{Mode: "all"},
	}
	result := mergeFieldPolicies(policies)
	assert.Equal(t, "all", result.Mode)
}

func TestMergeFieldPolicies_UnionOfAllowLists(t *testing.T) {
	// Role A allows field X, role B allows field Y → result has both X and Y
	policies := []FieldPolicy{
		{Mode: "allow", Fields: []string{"id", "name"}},
		{Mode: "allow", Fields: []string{"status"}},
	}
	result := mergeFieldPolicies(policies)
	assert.Equal(t, "allow", result.Mode)
	assert.ElementsMatch(t, []string{"id", "name", "status"}, result.Fields)
}

func TestMergeFieldPolicies_AllDeny(t *testing.T) {
	policies := []FieldPolicy{
		{Mode: "deny", Fields: []string{"secret"}},
	}
	result := mergeFieldPolicies(policies)
	assert.Equal(t, "deny", result.Mode)
}

func TestMergeFieldPolicies_AllowOverridesDeny(t *testing.T) {
	// If any role has an "allow" list, union is returned (not deny)
	policies := []FieldPolicy{
		{Mode: "deny", Fields: []string{"secret"}},
		{Mode: "allow", Fields: []string{"id"}},
	}
	result := mergeFieldPolicies(policies)
	assert.Equal(t, "allow", result.Mode)
	assert.Contains(t, result.Fields, "id")
}

// --- FilterResponse tests ---

func echoCtxWithPolicy(t *testing.T, policy FieldPolicy) echo.Context {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("rbac_field_policy", policy)
	return c
}

func TestFilterResponse_AllPolicy_PassThrough(t *testing.T) {
	c := echoCtxWithPolicy(t, AllFields())
	input := map[string]any{"id": "1", "secret": "s3cr3t"}
	result := FilterResponse(c, input)
	m, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "1", m["id"])
	assert.Equal(t, "s3cr3t", m["secret"])
}

func TestFilterResponse_AllowPolicy_StripsOtherFields(t *testing.T) {
	c := echoCtxWithPolicy(t, AllowFields("id", "name"))
	input := map[string]any{"id": "1", "name": "Alice", "secret": "s3cr3t"}
	result := FilterResponse(c, input)

	data, err := json.Marshal(result)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, "1", m["id"])
	assert.Equal(t, "Alice", m["name"])
	assert.NotContains(t, m, "secret")
}

func TestFilterResponse_DenyPolicy_StripsListedFields(t *testing.T) {
	c := echoCtxWithPolicy(t, DenyFields("secret"))
	input := map[string]any{"id": "1", "name": "Alice", "secret": "s3cr3t"}
	result := FilterResponse(c, input)

	data, err := json.Marshal(result)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, "1", m["id"])
	assert.Equal(t, "Alice", m["name"])
	assert.NotContains(t, m, "secret")
}

func TestFilterResponse_NoPolicySet_PassThrough(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No policy set in context

	input := map[string]any{"id": "1", "secret": "s3cr3t"}
	result := FilterResponse(c, input)
	m, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "s3cr3t", m["secret"])
}

// --- RequirePermission middleware tests ---

func TestRequirePermission_Forbidden(t *testing.T) {
	repo := &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, nil
		},
	}
	svc := NewService(&mockEventStore{}, repo)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "user1")

	mw := RequirePermission(svc, "orders", "read")
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequirePermission_Allowed_PolicyInContext(t *testing.T) {
	repo := &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"role_viewer"}, nil
		},
		GetPermissionsForRoleFn: func(_ context.Context, _ string) ([]PermissionReadModel, error) {
			return []PermissionReadModel{
				{Module: "orders", Action: "read", FieldMode: "allow", FieldList: []string{"id"}},
			}, nil
		},
	}
	svc := NewService(&mockEventStore{}, repo)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "user1")

	var capturedPolicy FieldPolicy
	mw := RequirePermission(svc, "orders", "read")
	handler := mw(func(c echo.Context) error {
		capturedPolicy, _ = c.Get("rbac_field_policy").(FieldPolicy)
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "allow", capturedPolicy.Mode)
	assert.Equal(t, []string{"id"}, capturedPolicy.Fields)
}

// --- CreateRole / event sourcing ---

func TestCreateRole_AppendsEvent(t *testing.T) {
	var appended []eventstore.Event
	store := &mockEventStore{
		AppendFn: func(_ context.Context, events []eventstore.Event) error {
			appended = events
			return nil
		},
	}
	svc := NewService(store, &mockRepo{})

	err := svc.CreateRole(context.Background(), CreateRoleCmd{
		Name:        "admin",
		Description: "Full access",
		Permissions: []Permission{SuperAdminPermission()},
		Actor:       "system",
	})
	require.NoError(t, err)
	require.Len(t, appended, 1)

	rc, ok := appended[0].(*RoleCreated)
	require.True(t, ok)
	assert.Equal(t, "role_admin", rc.AggregateID())
	assert.Equal(t, "admin", rc.Name)
	assert.Equal(t, "system", rc.Metadata()["user_id"])
}

// --- ModuleDefinition registry ---

func TestRegisterModule(t *testing.T) {
	RegisterModule(ModuleDefinition{
		Name:         "test_module",
		Fields:       []string{"id", "name"},
		DefaultPerms: FullCRUD("test_module", AllFields()),
	})
	mods := Modules()
	def, ok := mods["test_module"]
	require.True(t, ok)
	assert.Equal(t, "test_module", def.Name)
	assert.Len(t, def.DefaultPerms, 5)
}
