package seeder_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
)

// --- hand-rolled mock: eventstore.EventStore ---

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

// --- hand-rolled mock: rbac.ReadRepository ---

type mockRepo struct {
	GetRoleByIDFn           func(ctx context.Context, id string) (*rbac.RoleReadModel, error)
	GetRoleByNameFn         func(ctx context.Context, name string) (*rbac.RoleReadModel, error)
	ListRolesFn             func(ctx context.Context) ([]rbac.RoleReadModel, error)
	GetPermissionsForRoleFn func(ctx context.Context, roleID string) ([]rbac.PermissionReadModel, error)
	GetRolesForUserFn       func(ctx context.Context, userID string) ([]string, error)
}

func (m *mockRepo) GetRoleByID(ctx context.Context, id string) (*rbac.RoleReadModel, error) {
	if m.GetRoleByIDFn != nil {
		return m.GetRoleByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockRepo) GetRoleByName(ctx context.Context, name string) (*rbac.RoleReadModel, error) {
	if m.GetRoleByNameFn != nil {
		return m.GetRoleByNameFn(ctx, name)
	}
	return nil, nil
}
func (m *mockRepo) ListRoles(ctx context.Context) ([]rbac.RoleReadModel, error) {
	if m.ListRolesFn != nil {
		return m.ListRolesFn(ctx)
	}
	return nil, nil
}
func (m *mockRepo) GetPermissionsForRole(ctx context.Context, roleID string) ([]rbac.PermissionReadModel, error) {
	if m.GetPermissionsForRoleFn != nil {
		return m.GetPermissionsForRoleFn(ctx, roleID)
	}
	return nil, nil
}
func (m *mockRepo) GetRolesForUser(ctx context.Context, userID string) ([]string, error) {
	if m.GetRolesForUserFn != nil {
		return m.GetRolesForUserFn(ctx, userID)
	}
	return nil, nil
}

// --- hand-rolled mock: seeder.UserCreator ---

type mockUserCreator struct {
	CreateUserFn func(ctx context.Context, cmd seeder.CreateUserCmd) (string, error)
	GetByEmailFn func(ctx context.Context, email string) (*seeder.UserRecord, error)
	AssignRoleFn func(ctx context.Context, userID, roleID, actor string) error
}

func (m *mockUserCreator) CreateUser(ctx context.Context, cmd seeder.CreateUserCmd) (string, error) {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, cmd)
	}
	return "100", nil
}
func (m *mockUserCreator) GetByEmail(ctx context.Context, email string) (*seeder.UserRecord, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}
	return nil, nil
}
func (m *mockUserCreator) AssignRole(ctx context.Context, userID, roleID, actor string) error {
	if m.AssignRoleFn != nil {
		return m.AssignRoleFn(ctx, userID, roleID, actor)
	}
	return nil
}

// --- helpers ---

func newRbacService(store eventstore.EventStore, repo rbac.ReadRepository) *rbac.Service {
	if store == nil {
		store = &mockEventStore{}
	}
	if repo == nil {
		repo = &mockRepo{}
	}
	return rbac.NewService(store, repo)
}

// stubRbacSvc returns an rbac.Service whose GetRoleByName always returns a role
// with the given id. Use when role lookup must succeed but the exact ID doesn't matter.
func stubRbacSvc(id int64) *rbac.Service {
	return newRbacService(nil, &mockRepo{
		GetRoleByNameFn: func(_ context.Context, name string) (*rbac.RoleReadModel, error) {
			return &rbac.RoleReadModel{ID: id, Name: name}, nil
		},
	})
}

// registerTestModule registers a temporary module for testing. Caller must deregister
// if needed; in practice the registry is append-only so we use a unique name per test.
func registerTestModule(name string) {
	rbac.RegisterModule(rbac.ModuleDefinition{
		Name:         name,
		Fields:       []string{"id", "name"},
		DefaultPerms: rbac.FullCRUD(name, rbac.AllFields()),
	})
}

// --- SeedSuperAdminRole ---

func TestSeedSuperAdminRole_CreatesRoleWhenAbsent(t *testing.T) {
	var createCalled int
	store := &mockEventStore{
		AppendFn: func(_ context.Context, events []eventstore.Event) error {
			createCalled++
			return nil
		},
	}
	repo := &mockRepo{
		GetRoleByNameFn: func(_ context.Context, _ string) (*rbac.RoleReadModel, error) {
			return nil, nil // not found
		},
	}
	svc := newRbacService(store, repo)

	err := seeder.SeedSuperAdminRole(context.Background(), svc)
	require.NoError(t, err)
	assert.Equal(t, 1, createCalled, "CreateRole should be called exactly once")
}

func TestSeedSuperAdminRole_IdempotentWhenRoleExists(t *testing.T) {
	var createCalled int
	store := &mockEventStore{
		AppendFn: func(_ context.Context, _ []eventstore.Event) error {
			createCalled++
			return nil
		},
	}
	repo := &mockRepo{
		GetRoleByNameFn: func(_ context.Context, _ string) (*rbac.RoleReadModel, error) {
			return &rbac.RoleReadModel{ID: int64(1), Name: "super_admin"}, nil
		},
	}
	svc := newRbacService(store, repo)

	// Call twice — CreateRole must be called zero times total
	err := seeder.SeedSuperAdminRole(context.Background(), svc)
	require.NoError(t, err)
	err = seeder.SeedSuperAdminRole(context.Background(), svc)
	require.NoError(t, err)
	assert.Equal(t, 0, createCalled, "CreateRole must not be called when role already exists")
}

// --- SeedModuleRoles ---

func TestSeedModuleRoles_CreatesTwoRolesForTwoModules(t *testing.T) {
	// Register two unique module names for this test
	mod1 := "seedtest_alpha"
	mod2 := "seedtest_beta"
	registerTestModule(mod1)
	registerTestModule(mod2)

	var appendCalls int
	store := &mockEventStore{
		AppendFn: func(_ context.Context, events []eventstore.Event) error {
			if len(events) > 0 {
				appendCalls++
			}
			return nil
		},
	}
	repo := &mockRepo{
		// Return nil for both module roles (not yet seeded)
		GetRoleByNameFn: func(_ context.Context, _ string) (*rbac.RoleReadModel, error) {
			return nil, nil
		},
	}
	svc := newRbacService(store, repo)

	err := seeder.SeedModuleRoles(context.Background(), svc)
	require.NoError(t, err)
	// At minimum the two newly registered modules must each produce an Append call.
	assert.GreaterOrEqual(t, appendCalls, 2, "expected at least 2 CreateRole calls")
}

func TestSeedModuleRoles_SkipsExistingRoles(t *testing.T) {
	mod := "seedtest_gamma"
	registerTestModule(mod)

	var createCalled int
	store := &mockEventStore{
		AppendFn: func(_ context.Context, _ []eventstore.Event) error {
			createCalled++
			return nil
		},
	}
	repo := &mockRepo{
		// Report ALL roles as already existing so nothing is created
		GetRoleByNameFn: func(_ context.Context, _ string) (*rbac.RoleReadModel, error) {
			return &rbac.RoleReadModel{ID: int64(999), Name: mod + "_admin"}, nil
		},
	}
	svc := newRbacService(store, repo)

	err := seeder.SeedModuleRoles(context.Background(), svc)
	require.NoError(t, err)
	assert.Equal(t, 0, createCalled, "CreateRole must not be called when all roles already exist")
}

// --- SeedSuperAdminUser ---

func TestSeedSuperAdminUser_CreatesUserAndAssignsRole(t *testing.T) {
	const superAdminRoleID = int64(42)
	var createCalled, assignCalled int
	userSvc := &mockUserCreator{
		GetByEmailFn: func(_ context.Context, _ string) (*seeder.UserRecord, error) {
			return nil, nil // not found
		},
		CreateUserFn: func(_ context.Context, cmd seeder.CreateUserCmd) (string, error) {
			createCalled++
			assert.Equal(t, "admin@system.local", cmd.Email)
			assert.Equal(t, "system", cmd.Actor)
			return "200", nil
		},
		AssignRoleFn: func(_ context.Context, userID, roleID, actor string) error {
			assignCalled++
			assert.Equal(t, "200", userID)
			assert.Equal(t, strconv.FormatInt(superAdminRoleID, 10), roleID)
			assert.Equal(t, "system", actor)
			return nil
		},
	}
	repo := &mockRepo{
		GetRoleByNameFn: func(_ context.Context, name string) (*rbac.RoleReadModel, error) {
			if name == "super_admin" {
				return &rbac.RoleReadModel{ID: superAdminRoleID, Name: "super_admin"}, nil
			}
			return nil, nil
		},
	}
	svc := newRbacService(nil, repo)

	err := seeder.SeedSuperAdminUser(context.Background(), userSvc, svc, "s3cr3t!")
	require.NoError(t, err)
	assert.Equal(t, 1, createCalled)
	assert.Equal(t, 1, assignCalled)
}

func TestSeedSuperAdminUser_IdempotentWhenUserExists(t *testing.T) {
	var createCalled int
	userSvc := &mockUserCreator{
		GetByEmailFn: func(_ context.Context, _ string) (*seeder.UserRecord, error) {
			return &seeder.UserRecord{ID: "existing", Email: "admin@system.local"}, nil
		},
		CreateUserFn: func(_ context.Context, _ seeder.CreateUserCmd) (string, error) {
			createCalled++
			return "", nil
		},
	}
	svc := newRbacService(nil, nil)

	err := seeder.SeedSuperAdminUser(context.Background(), userSvc, svc, "s3cr3t!")
	require.NoError(t, err)
	assert.Equal(t, 0, createCalled)
}

// --- SeedModuleUsers ---

func TestSeedModuleUsers_UsesDefaultPasswordWhenEnvNotSet(t *testing.T) {
	mod := "seedtest_delta"
	registerTestModule(mod)

	passwords := map[string]string{}
	userSvc := &mockUserCreator{
		GetByEmailFn: func(_ context.Context, email string) (*seeder.UserRecord, error) {
			return nil, nil // none pre-existing
		},
		CreateUserFn: func(_ context.Context, cmd seeder.CreateUserCmd) (string, error) {
			passwords[cmd.Email] = cmd.Password
			return "300", nil
		},
	}
	rbacSvc := stubRbacSvc(int64(1))

	err := seeder.SeedModuleUsers(context.Background(), userSvc, rbacSvc, "defaultPass")
	require.NoError(t, err)
	assert.Equal(t, "defaultPass", passwords[mod+"_admin@system.local"])
}

func TestSeedModuleUsers_UsesEnvPasswordWhenSet(t *testing.T) {
	mod := "seedtest_epsilon"
	registerTestModule(mod)

	t.Setenv("SEED_SEEDTEST_EPSILON_ADMIN_PASSWORD", "envPassword!")

	passwords := map[string]string{}
	userSvc := &mockUserCreator{
		GetByEmailFn: func(_ context.Context, _ string) (*seeder.UserRecord, error) {
			return nil, nil
		},
		CreateUserFn: func(_ context.Context, cmd seeder.CreateUserCmd) (string, error) {
			passwords[cmd.Email] = cmd.Password
			return "301", nil
		},
	}
	rbacSvc := stubRbacSvc(int64(1))

	err := seeder.SeedModuleUsers(context.Background(), userSvc, rbacSvc, "defaultPass")
	require.NoError(t, err)
	assert.Equal(t, "envPassword!", passwords[mod+"_admin@system.local"])
}

func TestSeedModuleUsers_AssignsCorrectRoleID(t *testing.T) {
	mod := "seedtest_zeta"
	registerTestModule(mod)

	const roleID = int64(101)
	assignedRoles := map[string]string{} // userID -> roleID string
	userSvc := &mockUserCreator{
		GetByEmailFn: func(_ context.Context, _ string) (*seeder.UserRecord, error) {
			return nil, nil
		},
		CreateUserFn: func(_ context.Context, cmd seeder.CreateUserCmd) (string, error) {
			return "uid_" + cmd.Email, nil
		},
		AssignRoleFn: func(_ context.Context, userID, rid, _ string) error {
			assignedRoles[userID] = rid
			return nil
		},
	}
	rbacSvc := newRbacService(nil, &mockRepo{
		GetRoleByNameFn: func(_ context.Context, name string) (*rbac.RoleReadModel, error) {
			return &rbac.RoleReadModel{ID: roleID, Name: name}, nil
		},
	})

	err := seeder.SeedModuleUsers(context.Background(), userSvc, rbacSvc, "pw")
	require.NoError(t, err)
	assert.Equal(t,
		strconv.FormatInt(roleID, 10),
		assignedRoles["uid_"+mod+"_admin@system.local"],
	)
}

func TestSeedModuleUsers_SkipsExistingUsers(t *testing.T) {
	mod := "seedtest_eta"
	registerTestModule(mod)

	var createCalled int
	userSvc := &mockUserCreator{
		GetByEmailFn: func(_ context.Context, email string) (*seeder.UserRecord, error) {
			if email == mod+"_admin@system.local" {
				return &seeder.UserRecord{ID: "existing", Email: email}, nil
			}
			return nil, nil
		},
		CreateUserFn: func(_ context.Context, cmd seeder.CreateUserCmd) (string, error) {
			if cmd.Email == mod+"_admin@system.local" {
				createCalled++
			}
			return "uid", nil
		},
	}
	rbacSvc := stubRbacSvc(int64(1))

	err := seeder.SeedModuleUsers(context.Background(), userSvc, rbacSvc, "pw")
	require.NoError(t, err)
	assert.Equal(t, 0, createCalled, "CreateUser must not be called for existing user")
}
