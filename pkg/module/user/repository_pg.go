package user

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
)

type pgReadRepo struct{ db *sqlx.DB }

func NewPgReadRepository(db *sqlx.DB) ReadRepository { return &pgReadRepo{db: db} }

var allowedSorts = map[string]string{
	"email":      "email",
	"created_at": "created_at",
	"updated_at": "updated_at",
}

func (r *pgReadRepo) GetByID(ctx context.Context, id string) (*UserReadModel, error) {
	return database.Get[UserReadModel](ctx, r.db,
		`SELECT * FROM users_read WHERE id = $1`, id)
}

func (r *pgReadRepo) GetByEmail(ctx context.Context, email string) (*UserReadModel, error) {
	return database.Get[UserReadModel](ctx, r.db,
		`SELECT * FROM users_read WHERE email = $1`, email)
}

func (r *pgReadRepo) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	pr := database.PageRequest{Page: req.Page, PageSize: req.PageSize, SortBy: req.SortBy, SortDir: req.SortDir}
	pr.Normalise("created_at")
	page, err := database.PaginatedSelect[UserReadModel](ctx, r.db,
		`SELECT * FROM users_read`, pr, allowedSorts)
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Items: page.Items, Total: page.Total, Page: page.Page,
		PageSize: page.PageSize, TotalPages: page.TotalPages,
	}, nil
}
