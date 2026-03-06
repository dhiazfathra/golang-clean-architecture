package envvar

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByPlatformKey(ctx context.Context, platform, key string) (*EnvVar, error) {
	return database.Get[EnvVar](ctx, r.db,
		"SELECT * FROM env_vars WHERE platform = $1 AND key = $2", platform, key)
}

func (r *Repository) ListByPlatform(ctx context.Context, platform string, req database.PageRequest) (*database.PageResponse[EnvVar], error) {
	return database.PaginatedSelect[EnvVar](
		ctx, r.db,
		"SELECT * FROM env_vars WHERE platform = $1",
		req,
		map[string]string{"key": "key", "created_at": "created_at"},
		platform,
	)
}

func (r *Repository) Create(ctx context.Context, e *EnvVar) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO env_vars (id, platform, key, value, created_by, updated_by)
		 VALUES (:id, :platform, :key, :value, :created_by, :updated_by)`, e)
	return err
}

func (r *Repository) Update(ctx context.Context, e *EnvVar) error {
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE env_vars
		 SET value = :value, updated_at = NOW(), updated_by = :updated_by
		 WHERE id = :id AND NOT is_deleted`, e)
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64, deletedBy string) error {
	_, err := r.db.ExecContext(ctx, //nolint:gosec // G701: false positive, query uses parameterized placeholders
		`UPDATE env_vars SET is_deleted = true, updated_at = NOW(), updated_by = $1
		 WHERE id = $2 AND NOT is_deleted`, deletedBy, id)
	return err
}

func (r *Repository) ListAll(ctx context.Context) ([]EnvVar, error) {
	return database.Select[EnvVar](ctx, r.db,
		"SELECT * FROM env_vars ORDER BY platform, key")
}
