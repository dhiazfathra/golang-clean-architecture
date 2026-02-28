package database

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

type BaseReadModel struct {
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	CreatedBy string    `db:"created_by" json:"created_by"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	UpdatedBy string    `db:"updated_by" json:"updated_by"`
	IsDeleted bool      `db:"is_deleted" json:"-"`
}

// AuditMeta holds the timestamp and actor extracted from event metadata.
type AuditMeta struct {
	At time.Time
	By string
}

// UpsertReadModel executes an INSERT … ON CONFLICT DO UPDATE that preserves
// created_* on conflict and always updates updated_*.
// query must be a named query with :created_at, :created_by, :updated_at,
// :updated_by placeholders in the INSERT and EXCLUDED.updated_* in the UPDATE.
func UpsertReadModel(ctx context.Context, db *sqlx.DB, query string, arg any) error {
	_, err := db.NamedExecContext(ctx, query, arg)
	return err
}
