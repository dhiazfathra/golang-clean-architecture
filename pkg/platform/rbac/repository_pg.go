package rbac

import (
	"context"
	"strconv"

	"github.com/jmoiron/sqlx"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
)

type pgReadRepo struct{ db *sqlx.DB }

func NewPgReadRepository(db *sqlx.DB) ReadRepository { return &pgReadRepo{db: db} }

func (r *pgReadRepo) GetRoleByID(ctx context.Context, id string) (*RoleReadModel, error) {
	return database.Get[RoleReadModel](ctx, r.db,
		`SELECT * FROM roles_read WHERE id = $1`, id)
}

func (r *pgReadRepo) GetRoleByName(ctx context.Context, name string) (*RoleReadModel, error) {
	return database.Get[RoleReadModel](ctx, r.db,
		`SELECT * FROM roles_read WHERE name = $1`, name)
}

func (r *pgReadRepo) ListRoles(ctx context.Context) ([]RoleReadModel, error) {
	return database.Select[RoleReadModel](ctx, r.db,
		`SELECT * FROM roles_read ORDER BY name ASC`)
}

func (r *pgReadRepo) GetPermissionsForRole(ctx context.Context, roleID string) ([]PermissionReadModel, error) {
	return database.Select[PermissionReadModel](ctx, r.db,
		`SELECT * FROM permissions_read WHERE role_id = $1`, roleID)
}

func (r *pgReadRepo) GetRolesForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := database.Select[UserRoleReadModel](ctx, r.db,
		`SELECT * FROM user_roles_read WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = strconv.FormatInt(row.RoleID, 10)
	}
	return ids, nil
}
