package user

import "context"

type ReadRepository interface {
	GetByID(ctx context.Context, id string) (*UserReadModel, error)
	GetByEmail(ctx context.Context, email string) (*UserReadModel, error)
	List(ctx context.Context, req ListRequest) (*ListResponse, error)
}

type ListRequest struct {
	Page, PageSize int
	SortBy, SortDir string
}

type ListResponse struct {
	Items      []UserReadModel
	Total, Page, PageSize, TotalPages int
}
