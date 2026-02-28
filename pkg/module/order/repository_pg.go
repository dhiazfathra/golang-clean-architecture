package order

import (
	"context"
	"strconv"

	"github.com/jmoiron/sqlx"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
)

type pgReadRepo struct{ db *sqlx.DB }

func NewPgReadRepository(db *sqlx.DB) ReadRepository { return &pgReadRepo{db: db} }

var allowedSorts = map[string]string{
	"created_at": "created_at",
	"total":      "total",
	"status":     "status",
}

func (r *pgReadRepo) GetByID(ctx context.Context, id string) (*OrderReadModel, error) {
	orderID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, err
	}
	return database.Get[OrderReadModel](ctx, r.db,
		`SELECT * FROM orders_read WHERE id = $1`, orderID)
}

func (r *pgReadRepo) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	pr := database.PageRequest{Page: req.Page, PageSize: req.PageSize, SortBy: req.SortBy, SortDir: req.SortDir}
	pr.Normalise("created_at")
	page, err := database.PaginatedSelect[OrderReadModel](ctx, r.db,
		`SELECT * FROM orders_read`, pr, allowedSorts)
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Items: page.Items, Total: page.Total, Page: page.Page,
		PageSize: page.PageSize, TotalPages: page.TotalPages,
	}, nil
}
