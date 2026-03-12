package rbac

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/observability"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func init() {
	observability.InitNoop()
}

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
	GetRoleByIDFn           func(ctx context.Context, id string) (*RoleReadModel, error)
	GetRoleByNameFn         func(ctx context.Context, name string) (*RoleReadModel, error)
	ListRolesFn             func(ctx context.Context) ([]RoleReadModel, error)
	GetPermissionsForRoleFn func(ctx context.Context, roleID string) ([]PermissionReadModel, error)
	GetRolesForUserFn       func(ctx context.Context, userID string) ([]string, error)
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
	_, parseErr := strconv.ParseInt(rc.AggregateID(), 10, 64)
	assert.NoError(t, parseErr, "aggregate ID must be a valid snowflake decimal string")
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

// --- applyRole tests ---

func TestApplyRole_RoleCreated(t *testing.T) {
	agg := newRoleAggregate("r1")
	e := &RoleCreated{
		BaseEvent:   eventstore.NewBaseEvent("r1", "role", "role.created", 1, nil),
		Name:        "admin",
		Description: "Full access",
		Permissions: []Permission{SuperAdminPermission()},
	}
	agg.Apply(e)
	assert.Equal(t, "r1", agg.State.ID)
	assert.Equal(t, "admin", agg.State.Name)
	assert.True(t, agg.State.Active)
	assert.Len(t, agg.State.Permissions, 1)
}

func TestApplyRole_PermissionGranted(t *testing.T) {
	agg := newRoleAggregate("r1")
	agg.Apply(&RoleCreated{
		BaseEvent: eventstore.NewBaseEvent("r1", "role", "role.created", 1, nil),
		Name:      "editor",
	})
	agg.Apply(&PermissionGranted{
		BaseEvent:  eventstore.NewBaseEvent("r1", "role", "role.permission_granted", 2, nil),
		Permission: Permission{Module: "users", Action: "read", Fields: AllFields()},
	})
	assert.Len(t, agg.State.Permissions, 1)
	assert.Equal(t, "users", agg.State.Permissions[0].Module)
}

func TestApplyRole_PermissionRevoked(t *testing.T) {
	agg := newRoleAggregate("r1")
	agg.Apply(&RoleCreated{
		BaseEvent:   eventstore.NewBaseEvent("r1", "role", "role.created", 1, nil),
		Name:        "editor",
		Permissions: []Permission{{Module: "users", Action: "read"}, {Module: "users", Action: "write"}},
	})
	agg.Apply(&PermissionRevoked{
		BaseEvent: eventstore.NewBaseEvent("r1", "role", "role.permission_revoked", 2, nil),
		Module:    "users",
		Action:    "read",
	})
	assert.Len(t, agg.State.Permissions, 1)
	assert.Equal(t, "write", agg.State.Permissions[0].Action)
}

func TestApplyRole_RoleDeleted(t *testing.T) {
	agg := newRoleAggregate("r1")
	agg.Apply(&RoleCreated{
		BaseEvent: eventstore.NewBaseEvent("r1", "role", "role.created", 1, nil),
		Name:      "temp",
	})
	assert.True(t, agg.State.Active)
	agg.Apply(&RoleDeleted{
		BaseEvent: eventstore.NewBaseEvent("r1", "role", "role.deleted", 2, nil),
	})
	assert.False(t, agg.State.Active)
}

func TestApplyRole_UnknownEvent(t *testing.T) {
	agg := newRoleAggregate("r1")
	e := eventstore.NewBaseEvent("r1", "role", "unknown", 1, nil)
	agg.Apply(&e) // should not panic
	assert.Equal(t, RoleState{}, agg.State)
}

// --- Model helpers ---

func TestAllFields(t *testing.T) {
	fp := AllFields()
	assert.Equal(t, "all", fp.Mode)
	assert.Nil(t, fp.Fields)
}

func TestAllowFields(t *testing.T) {
	fp := AllowFields("id", "name")
	assert.Equal(t, "allow", fp.Mode)
	assert.Equal(t, []string{"id", "name"}, fp.Fields)
}

func TestDenyFields(t *testing.T) {
	fp := DenyFields("secret")
	assert.Equal(t, "deny", fp.Mode)
	assert.Equal(t, []string{"secret"}, fp.Fields)
}

func TestFullCRUD(t *testing.T) {
	perms := FullCRUD("orders", AllFields())
	assert.Len(t, perms, 5)
	actions := []string{"create", "read", "update", "delete", "list"}
	for i, p := range perms {
		assert.Equal(t, "orders", p.Module)
		assert.Equal(t, actions[i], p.Action)
		assert.Equal(t, "all", p.Fields.Mode)
	}
}

func TestSuperAdminPermission(t *testing.T) {
	p := SuperAdminPermission()
	assert.Equal(t, "*", p.Module)
	assert.Equal(t, "*", p.Action)
	assert.Equal(t, "all", p.Fields.Mode)
}

// --- Service additional tests ---

func TestDeleteRole(t *testing.T) {
	store := &mockEventStore{
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return []eventstore.Event{
				&RoleCreated{BaseEvent: eventstore.NewBaseEvent("r1", "role", "role.created", 1, nil), Name: "temp"},
			}, nil
		},
	}
	svc := NewService(store, &mockRepo{})
	err := svc.DeleteRole(context.Background(), "r1", "admin")
	assert.NoError(t, err)
}

func TestDeleteRoleLoadError(t *testing.T) {
	store := &mockEventStore{
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return nil, assert.AnError
		},
	}
	svc := NewService(store, &mockRepo{})
	err := svc.DeleteRole(context.Background(), "r1", "admin")
	assert.Error(t, err)
}

func TestGrantPermission(t *testing.T) {
	store := &mockEventStore{
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return []eventstore.Event{
				&RoleCreated{BaseEvent: eventstore.NewBaseEvent("r1", "role", "role.created", 1, nil), Name: "editor"},
			}, nil
		},
	}
	svc := NewService(store, &mockRepo{})
	err := svc.GrantPermission(context.Background(), "r1", Permission{Module: "users", Action: "read"}, "admin")
	assert.NoError(t, err)
}

func TestGrantPermissionLoadError(t *testing.T) {
	store := &mockEventStore{
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return nil, assert.AnError
		},
	}
	svc := NewService(store, &mockRepo{})
	err := svc.GrantPermission(context.Background(), "r1", Permission{}, "admin")
	assert.Error(t, err)
}

func TestRevokePermission(t *testing.T) {
	store := &mockEventStore{
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return []eventstore.Event{
				&RoleCreated{BaseEvent: eventstore.NewBaseEvent("r1", "role", "role.created", 1, nil), Name: "editor"},
			}, nil
		},
	}
	svc := NewService(store, &mockRepo{})
	err := svc.RevokePermission(context.Background(), "r1", "users", "read", "admin")
	assert.NoError(t, err)
}

func TestRevokePermissionLoadError(t *testing.T) {
	store := &mockEventStore{
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return nil, assert.AnError
		},
	}
	svc := NewService(store, &mockRepo{})
	err := svc.RevokePermission(context.Background(), "r1", "users", "read", "admin")
	assert.Error(t, err)
}

func TestAssignRoleStub(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{})
	err := svc.AssignRole(context.Background(), "u1", "admin", "system")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be called through user.Service")
}

func TestGetRoleByID(t *testing.T) {
	repo := &mockRepo{
		GetRoleByIDFn: func(_ context.Context, _ string) (*RoleReadModel, error) {
			return &RoleReadModel{ID: 1, Name: "admin"}, nil
		},
	}
	svc := NewService(&mockEventStore{}, repo)
	role, err := svc.GetRoleByID(context.Background(), "1")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)
}

func TestGetRoleByName(t *testing.T) {
	repo := &mockRepo{
		GetRoleByNameFn: func(_ context.Context, _ string) (*RoleReadModel, error) {
			return &RoleReadModel{ID: 1, Name: "admin"}, nil
		},
	}
	svc := NewService(&mockEventStore{}, repo)
	role, err := svc.GetRoleByName(context.Background(), "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)
}

func TestGetRolesForUser(t *testing.T) {
	repo := &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"r1", "r2"}, nil
		},
	}
	svc := NewService(&mockEventStore{}, repo)
	roles, err := svc.GetRolesForUser(context.Background(), "u1")
	require.NoError(t, err)
	assert.Len(t, roles, 2)
}

func TestCheckPermissionGetRolesError(t *testing.T) {
	repo := &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, assert.AnError
		},
	}
	svc := NewService(&mockEventStore{}, repo)
	allowed, _, err := svc.CheckPermission(context.Background(), "u1", "orders", "read")
	assert.Error(t, err)
	assert.False(t, allowed)
}

func TestCheckPermissionGetPermsError(t *testing.T) {
	repo := &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"r1"}, nil
		},
		GetPermissionsForRoleFn: func(_ context.Context, _ string) ([]PermissionReadModel, error) {
			return nil, assert.AnError
		},
	}
	svc := NewService(&mockEventStore{}, repo)
	allowed, _, err := svc.CheckPermission(context.Background(), "u1", "orders", "read")
	assert.Error(t, err)
	assert.False(t, allowed)
}

// --- RequirePermission middleware error path ---

func TestRequirePermissionCheckError(t *testing.T) {
	repo := &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, assert.AnError
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
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- ValidateInputFields ---

func TestValidateInputFieldsAllPolicy(t *testing.T) {
	c := echoCtxWithPolicy(t, AllFields())
	result := ValidateInputFields(c, map[string]any{"id": 1, "secret": "x"})
	assert.Nil(t, result)
}

func TestValidateInputFieldsNoPolicy(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	result := ValidateInputFields(c, map[string]any{"id": 1})
	assert.Nil(t, result)
}

func TestValidateInputFieldsAllowPolicy(t *testing.T) {
	c := echoCtxWithPolicy(t, AllowFields("id", "name"))
	result := ValidateInputFields(c, map[string]any{"id": 1, "name": "Alice", "secret": "x"})
	assert.Contains(t, result, "secret")
}

func TestValidateInputFieldsDenyPolicy(t *testing.T) {
	c := echoCtxWithPolicy(t, DenyFields("secret"))
	result := ValidateInputFields(c, map[string]any{"id": 1, "secret": "x"})
	assert.Contains(t, result, "secret")
}

// --- FilterResponse with PageResponse ---

func TestFilterResponsePageResponse(t *testing.T) {
	c := echoCtxWithPolicy(t, AllowFields("id"))
	input := map[string]any{
		"items": []map[string]any{
			{"id": "1", "secret": "s"},
			{"id": "2", "secret": "s"},
		},
		"total":       2,
		"page":        1,
		"page_size":   10,
		"total_pages": 1,
	}
	result := FilterResponse(c, input)
	data, _ := json.Marshal(result)
	assert.NotContains(t, string(data), "secret")
	assert.Contains(t, string(data), "id")
}

func TestFilterMapUnmarshalError(t *testing.T) {
	result := filterMap([]byte(`{invalid`), FieldPolicy{Mode: "allow"})
	assert.Nil(t, result)
}

// --- RegisterRoutes ---

func TestRegisterRoutes(t *testing.T) {
	e := echo.New()
	g := e.Group("/admin")
	svc := NewService(&mockEventStore{}, &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, nil
		},
	})
	h := NewHandler(svc, nil)
	RegisterRoutes(g, h, svc)
	assert.True(t, len(e.Routes()) >= 8)
}

// --- Projector tests ---

func TestNewProjector(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	p := NewProjector(db)
	assert.NotNil(t, p)
	assert.Equal(t, "rbac", p.Name())
}

func TestProjectorHandleRoleCreated(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)

	mock.ExpectExec("INSERT INTO roles_read").WillReturnResult(sqlmock.NewResult(1, 1))

	e := &RoleCreated{
		BaseEvent:   eventstore.NewBaseEvent("123", "role", "role.created", 1, map[string]string{"user_id": "admin"}),
		Name:        "editor",
		Description: "Can edit",
	}
	err := p.Handle(context.Background(), e)
	assert.NoError(t, err)
}

func TestProjectorHandlePermissionGranted(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)

	mock.ExpectExec("INSERT INTO permissions_read").WillReturnResult(sqlmock.NewResult(1, 1))

	e := &PermissionGranted{
		BaseEvent:  eventstore.NewBaseEvent("123", "role", "role.permission_granted", 2, map[string]string{"user_id": "admin"}),
		Permission: Permission{Module: "users", Action: "read", Fields: AllFields()},
	}
	err := p.Handle(context.Background(), e)
	assert.NoError(t, err)
}

func TestProjectorHandlePermissionRevoked(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)

	mock.ExpectExec("UPDATE permissions_read").WillReturnResult(sqlmock.NewResult(0, 1))

	e := &PermissionRevoked{
		BaseEvent: eventstore.NewBaseEvent("123", "role", "role.permission_revoked", 3, map[string]string{"user_id": "admin"}),
		Module:    "users",
		Action:    "read",
	}
	err := p.Handle(context.Background(), e)
	assert.NoError(t, err)
}

func TestProjectorHandleRoleDeleted(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	p := NewProjector(db)

	mock.ExpectExec("UPDATE roles_read").WillReturnResult(sqlmock.NewResult(0, 1))

	e := &RoleDeleted{
		BaseEvent: eventstore.NewBaseEvent("123", "role", "role.deleted", 4, map[string]string{"user_id": "admin"}),
	}
	err := p.Handle(context.Background(), e)
	assert.NoError(t, err)
}

func TestProjectorHandleUnknownEvent(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	p := NewProjector(db)

	e := eventstore.NewBaseEvent("123", "role", "unknown", 1, nil)
	err := p.Handle(context.Background(), e)
	assert.NoError(t, err)
}

// --- PgReadRepository tests ---

func TestNewPgReadRepository(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	repo := NewPgReadRepository(db)
	assert.NotNil(t, repo)
}

func TestPgRepoGetRoleByID(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewPgReadRepository(db)
	now := time.Now()

	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "created_at", "created_by", "updated_at", "updated_by", "is_deleted"}).
			AddRow(int64(1), "admin", "Full access", now, "sys", now, "sys", false))

	role, err := repo.GetRoleByID(context.Background(), "1")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)
}

func TestPgRepoGetRoleByName(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewPgReadRepository(db)
	now := time.Now()

	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "created_at", "created_by", "updated_at", "updated_by", "is_deleted"}).
			AddRow(int64(1), "admin", "Full access", now, "sys", now, "sys", false))

	role, err := repo.GetRoleByName(context.Background(), "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)
}

func TestPgRepoListRoles(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewPgReadRepository(db)
	now := time.Now()

	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "created_at", "created_by", "updated_at", "updated_by", "is_deleted"}).
			AddRow(int64(1), "admin", "desc", now, "sys", now, "sys", false))

	roles, err := repo.ListRoles(context.Background())
	require.NoError(t, err)
	assert.Len(t, roles, 1)
}

func TestPgRepoGetPermissionsForRole(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewPgReadRepository(db)
	now := time.Now()

	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "role_id", "module", "action", "field_mode", "field_list", "created_at", "created_by", "updated_at", "updated_by", "is_deleted"}).
			AddRow(int64(1), int64(10), "users", "read", "all", "{}", now, "sys", now, "sys", false))

	perms, err := repo.GetPermissionsForRole(context.Background(), "10")
	require.NoError(t, err)
	assert.Len(t, perms, 1)
}

func TestPgRepoGetRolesForUser(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewPgReadRepository(db)
	now := time.Now()

	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "role_id", "created_at", "created_by", "updated_at", "updated_by", "is_deleted"}).
			AddRow(int64(1), int64(10), now, "sys", now, "sys", false).
			AddRow(int64(1), int64(20), now, "sys", now, "sys", false))

	ids, err := repo.GetRolesForUser(context.Background(), "1")
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Equal(t, "10", ids[0])
	assert.Equal(t, "20", ids[1])
}

func TestPgRepoGetRolesForUserError(t *testing.T) {
	db, mock := testutil.NewMockDB(t)
	repo := NewPgReadRepository(db)

	mock.ExpectQuery("SELECT").WillReturnError(assert.AnError)

	_, err := repo.GetRolesForUser(context.Background(), "1")
	assert.Error(t, err)
}

// --- Handler error path tests ---

func TestHandlerCreateRoleServiceError(t *testing.T) {
	store := &mockEventStore{
		AppendFn: func(_ context.Context, _ []eventstore.Event) error {
			return assert.AnError
		},
	}
	svc := NewService(store, &mockRepo{})
	h := NewHandler(svc, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodPost, "/admin/roles",
		`{"name":"editor","description":"test"}`, AllFields())

	require.NoError(t, h.CreateRole(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandlerCreateRoleBindError(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{})
	h := NewHandler(svc, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodPost, "/admin/roles", "", AllFields())
	// Set invalid content type to trigger bind error
	c.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	require.NoError(t, h.CreateRole(c))
	// Empty body with JSON content type will bind to zero-value struct — name will be empty
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandlerListRolesError(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{
		ListRolesFn: func(_ context.Context) ([]RoleReadModel, error) {
			return nil, assert.AnError
		},
	})
	h := NewHandler(svc, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodGet, "/admin/roles", "", AllFields())

	require.NoError(t, h.ListRoles(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandlerGetRoleError(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{
		GetRoleByIDFn: func(_ context.Context, _ string) (*RoleReadModel, error) {
			return nil, assert.AnError
		},
	})
	h := NewHandler(svc, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodGet, "/admin/roles/1", "", AllFields())
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.GetRole(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandlerDeleteRoleError(t *testing.T) {
	store := &mockEventStore{
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return nil, assert.AnError
		},
	}
	svc := NewService(store, &mockRepo{})
	h := NewHandler(svc, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodDelete, "/admin/roles/1", "", AllFields())
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.DeleteRole(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandlerGrantPermissionError(t *testing.T) {
	store := &mockEventStore{
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return nil, assert.AnError
		},
	}
	svc := NewService(store, &mockRepo{})
	h := NewHandler(svc, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodPost, "/admin/roles/1/permissions",
		`{"module":"orders","action":"read","fields":{"mode":"all"}}`, AllFields())
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.GrantPermission(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandlerRevokePermissionError(t *testing.T) {
	store := &mockEventStore{
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return nil, assert.AnError
		},
	}
	svc := NewService(store, &mockRepo{})
	h := NewHandler(svc, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodDelete, "/admin/roles/1/permissions/orders:read", "", AllFields())
	c.SetParamNames("id", "perm")
	c.SetParamValues("1", "orders:read")

	require.NoError(t, h.RevokePermission(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandlerListUserRolesError(t *testing.T) {
	svc := NewService(&mockEventStore{}, &mockRepo{
		GetRolesForUserFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, assert.AnError
		},
	})
	h := NewHandler(svc, nil)
	c, rec := testutil.AuthedEchoCtxWithPolicy(http.MethodGet, "/admin/users/u1/roles", "", AllFields())
	c.SetParamNames("id")
	c.SetParamValues("u1")

	require.NoError(t, h.ListUserRoles(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- filterByPolicy marshal error path ---

func TestFilterByPolicyMarshalError(t *testing.T) {
	// Pass an unmarshalable value
	result := filterByPolicy(make(chan int), FieldPolicy{Mode: "allow", Fields: []string{"id"}})
	// Should return original value on marshal error
	assert.NotNil(t, result)
}
