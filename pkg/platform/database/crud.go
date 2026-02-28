package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// Get wraps the query to auto-filter soft-deleted rows.
func Get[T any](ctx context.Context, db *sqlx.DB, query string, args ...any) (*T, error) {
	wrapped := "SELECT * FROM (" + query + ") AS _t WHERE _t.is_deleted = false"
	var result T
	if err := db.GetContext(ctx, &result, wrapped, args...); err != nil {
		return nil, err
	}
	return &result, nil
}

// Select wraps the query to auto-filter soft-deleted rows.
func Select[T any](ctx context.Context, db *sqlx.DB, query string, args ...any) ([]T, error) {
	wrapped := "SELECT * FROM (" + query + ") AS _t WHERE _t.is_deleted = false"
	var results []T
	if err := db.SelectContext(ctx, &results, wrapped, args...); err != nil {
		return nil, err
	}
	return results, nil
}

func GetIncludingDeleted[T any](ctx context.Context, db *sqlx.DB, query string, args ...any) (*T, error) {
	var result T
	if err := db.GetContext(ctx, &result, query, args...); err != nil {
		return nil, err
	}
	return &result, nil
}

func SelectIncludingDeleted[T any](ctx context.Context, db *sqlx.DB, query string, args ...any) ([]T, error) {
	var results []T
	if err := db.SelectContext(ctx, &results, query, args...); err != nil {
		return nil, err
	}
	return results, nil
}

func Exec(ctx context.Context, db *sqlx.DB, query string, args ...any) error {
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func NamedExec(ctx context.Context, db *sqlx.DB, query string, arg any) error {
	_, err := db.NamedExecContext(ctx, query, arg)
	return err
}
