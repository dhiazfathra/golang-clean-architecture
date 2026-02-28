package rbac

import (
	"context"
	"fmt"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
)

type CreateRoleCmd struct {
	Name        string
	Description string
	Permissions []Permission
	Actor       string
}

type Service struct {
	store eventstore.EventStore
	repo  ReadRepository
}

func NewService(store eventstore.EventStore, repo ReadRepository) *Service {
	return &Service{store: store, repo: repo}
}

func (s *Service) CreateRole(ctx context.Context, cmd CreateRoleCmd) error {
	id := "role_" + cmd.Name
	meta := map[string]string{"user_id": cmd.Actor}
	agg := newRoleAggregate(id)
	e := &RoleCreated{
		BaseEvent:   eventstore.NewBaseEvent(id, "role", "role.created", 1, meta),
		Name:        cmd.Name,
		Description: cmd.Description,
		Permissions: cmd.Permissions,
	}
	agg.Apply(e)
	return s.store.Append(ctx, agg.Uncommitted())
}

func (s *Service) GetRoleByID(ctx context.Context, id string) (*RoleReadModel, error) {
	return s.repo.GetRoleByID(ctx, id)
}

func (s *Service) GetRoleByName(ctx context.Context, name string) (*RoleReadModel, error) {
	return s.repo.GetRoleByName(ctx, name)
}

func (s *Service) ListRoles(ctx context.Context) ([]RoleReadModel, error) {
	return s.repo.ListRoles(ctx)
}

func (s *Service) DeleteRole(ctx context.Context, roleID, actor string) error {
	agg, err := s.loadRole(ctx, roleID)
	if err != nil {
		return err
	}
	meta := map[string]string{"user_id": actor}
	e := &RoleDeleted{
		BaseEvent: eventstore.NewBaseEvent(roleID, "role", "role.deleted", agg.Version+1, meta),
	}
	agg.Apply(e)
	return s.store.Append(ctx, agg.Uncommitted())
}

func (s *Service) GrantPermission(ctx context.Context, roleID string, perm Permission, actor string) error {
	agg, err := s.loadRole(ctx, roleID)
	if err != nil {
		return err
	}
	meta := map[string]string{"user_id": actor}
	e := &PermissionGranted{
		BaseEvent:  eventstore.NewBaseEvent(roleID, "role", "role.permission_granted", agg.Version+1, meta),
		Permission: perm,
	}
	agg.Apply(e)
	return s.store.Append(ctx, agg.Uncommitted())
}

func (s *Service) RevokePermission(ctx context.Context, roleID, module, action, actor string) error {
	agg, err := s.loadRole(ctx, roleID)
	if err != nil {
		return err
	}
	meta := map[string]string{"user_id": actor}
	e := &PermissionRevoked{
		BaseEvent: eventstore.NewBaseEvent(roleID, "role", "role.permission_revoked", agg.Version+1, meta),
		Module:    module,
		Action:    action,
	}
	agg.Apply(e)
	return s.store.Append(ctx, agg.Uncommitted())
}

func (s *Service) GetRolesForUser(ctx context.Context, userID string) ([]string, error) {
	return s.repo.GetRolesForUser(ctx, userID)
}

// AssignRole is a stub — real implementation in M5 via user.Service.
func (s *Service) AssignRole(ctx context.Context, userID, roleName, actor string) error {
	return fmt.Errorf("rbac: AssignRole must be called through user.Service")
}

// CheckPermission returns (allowed, fieldPolicy, error).
func (s *Service) CheckPermission(ctx context.Context, userID, module, action string) (bool, FieldPolicy, error) {
	roleIDs, err := s.repo.GetRolesForUser(ctx, userID)
	if err != nil {
		return false, FieldPolicy{}, err
	}
	var policies []FieldPolicy
	for _, rid := range roleIDs {
		perms, err := s.repo.GetPermissionsForRole(ctx, rid)
		if err != nil {
			return false, FieldPolicy{}, err
		}
		for _, p := range perms {
			if matchesModuleAction(p, module, action) {
				policies = append(policies, FieldPolicy{Mode: p.FieldMode, Fields: p.FieldList})
			}
		}
	}
	if len(policies) == 0 {
		return false, FieldPolicy{}, nil
	}
	return true, mergeFieldPolicies(policies), nil
}

func (s *Service) loadRole(ctx context.Context, id string) (*eventstore.Aggregate[RoleState], error) {
	events, err := s.store.Load(ctx, "role", id, 0)
	if err != nil {
		return nil, fmt.Errorf("rbac: load role %s: %w", id, err)
	}
	agg := newRoleAggregate(id)
	for _, e := range events {
		agg.Rehydrate(e)
	}
	return agg, nil
}

func matchesModuleAction(p PermissionReadModel, module, action string) bool {
	modMatch := p.Module == "*" || p.Module == module
	actMatch := p.Action == "*" || p.Action == action
	return modMatch && actMatch
}

func mergeFieldPolicies(policies []FieldPolicy) FieldPolicy {
	for _, p := range policies {
		if p.Mode == "all" {
			return AllFields()
		}
	}
	// Union of allow lists
	seen := map[string]bool{}
	for _, p := range policies {
		if p.Mode == "allow" {
			for _, f := range p.Fields {
				seen[f] = true
			}
		}
	}
	if len(seen) > 0 {
		fields := make([]string, 0, len(seen))
		for f := range seen {
			fields = append(fields, f)
		}
		return AllowFields(fields...)
	}
	// All deny — intersection
	return DenyFields()
}
