package order

import "context"

type ReadRepository interface {
	GetByID(ctx context.Context, id string) (*OrderReadModel, error)
	List(ctx context.Context, req ListRequest) (*ListResponse, error)
}

type ListRequest struct {
	Page, PageSize  int
	SortBy, SortDir string
}

type ListResponse struct {
	Items                         []OrderReadModel
	Total, Page, PageSize, TotalPages int
}
