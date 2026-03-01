package featureflag

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

func (r *Repository) GetByKey(ctx context.Context, key string) (*Flag, error) {
	return database.Get[Flag](ctx, r.db,
		"SELECT * FROM feature_flags WHERE key = $1", key)
}

func (r *Repository) List(ctx context.Context) ([]Flag, error) {
	return database.Select[Flag](ctx, r.db,
		"SELECT * FROM feature_flags ORDER BY key ASC")
}

func (r *Repository) Create(ctx context.Context, f *Flag) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO feature_flags (id, key, enabled, description, metadata, created_by, updated_by)
		 VALUES (:id, :key, :enabled, :description, :metadata, :created_by, :updated_by)`, f)
	return err
}

func (r *Repository) Update(ctx context.Context, f *Flag) error {
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE feature_flags
		 SET enabled = :enabled, description = :description, metadata = :metadata,
		     updated_at = NOW(), updated_by = :updated_by
		 WHERE id = :id AND NOT is_deleted`, f)
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64, deletedBy string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE feature_flags SET is_deleted = true, updated_at = NOW(), updated_by = $1
		 WHERE id = $2 AND NOT is_deleted`, deletedBy, id)
	return err
}
