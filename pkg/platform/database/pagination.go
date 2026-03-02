package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type PageRequest struct {
	Page     int    `query:"page"`
	PageSize int    `query:"page_size"`
	SortBy   string `query:"sort_by"`
	SortDir  string `query:"sort_dir"`
}

func (r *PageRequest) Normalise(defaultSort string) {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		r.PageSize = 20
	}
	if r.SortDir != "desc" {
		r.SortDir = "asc"
	}
	if r.SortBy == "" {
		r.SortBy = defaultSort
	}
}

type PageResponse[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedSelect runs a count + paginated SELECT with is_deleted filtering.
// allowedSorts maps query param name → SQL column name (SQL-injection-safe).
func PaginatedSelect[T any](
	ctx context.Context,
	db *sqlx.DB,
	baseQuery string,
	req PageRequest,
	allowedSorts map[string]string,
	args ...any,
) (*PageResponse[T], error) {
	col, ok := allowedSorts[req.SortBy]
	if !ok {
		for _, v := range allowedSorts {
			col = v
			break
		}
	}
	filtered := "SELECT * FROM (" + baseQuery + ") AS _t WHERE _t.is_deleted = false"
	var total int64
	if err := db.GetContext(ctx, &total, "SELECT COUNT(*) FROM ("+filtered+") AS _c", args...); err != nil {
		return nil, err
	}
	offset := (req.Page - 1) * req.PageSize
	ordered := fmt.Sprintf("%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		filtered, col, req.SortDir, len(args)+1, len(args)+2)
	args = append(args, req.PageSize, offset)
	var items []T
	if err := db.SelectContext(ctx, &items, ordered, args...); err != nil {
		return nil, err
	}
	pageSize := int64(req.PageSize)
	totalPages := int((total + pageSize - 1) / pageSize)
	return &PageResponse[T]{
		Items: items, Total: total, Page: req.Page,
		PageSize: req.PageSize, TotalPages: totalPages,
	}, nil
}
