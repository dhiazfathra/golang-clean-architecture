package apitoken

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

func (r *Repository) Create(ctx context.Context, token *APIToken) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO api_tokens (id, name, token_hash, token_prefix, user_id, expires_at, created_by, updated_by)
		 VALUES (:id, :name, :token_hash, :token_prefix, :user_id, :expires_at, :created_by, :updated_by)`, token)
	return err
}

func (r *Repository) GetByHash(ctx context.Context, hash string) (*APIToken, error) {
	return database.Get[APIToken](ctx, r.db,
		"SELECT * FROM api_tokens WHERE token_hash = $1", hash)
}

func (r *Repository) ListByUser(ctx context.Context, userID string) ([]APIToken, error) {
	return database.Select[APIToken](ctx, r.db,
		"SELECT * FROM api_tokens WHERE user_id = $1 AND expires_at > NOW() ORDER BY created_at DESC", userID)
}

func (r *Repository) Delete(ctx context.Context, id int64, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE api_tokens SET is_deleted = true, updated_at = NOW(), updated_by = $1
		 WHERE id = $2 AND user_id = $1 AND NOT is_deleted`, userID, id)
	return err
}

func (r *Repository) LoadAll(ctx context.Context) (map[string]string, error) {
	type row struct {
		TokenHash string `db:"token_hash"`
		UserID    string `db:"user_id"`
	}
	var rows []row
	err := r.db.SelectContext(ctx,
		&rows,
		`SELECT token_hash, user_id FROM api_tokens WHERE NOT is_deleted AND expires_at > NOW()`)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.TokenHash] = r.UserID
	}
	return m, nil
}
