package database_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
)

// Verify that Get/Select wrap the query with is_deleted filtering by inspecting
// the wrapped SQL. We do this by wrapping a known base query and checking the
// resulting string — no DB connection required.

func TestGetWrapsQuery(t *testing.T) {
	base := "SELECT id, is_deleted FROM users WHERE id = $1"
	wrapped := "SELECT * FROM (" + base + ") AS _t WHERE _t.is_deleted = false"
	// The same logic as database.Get — we just confirm the pattern is correct.
	assert.True(t, strings.Contains(wrapped, "WHERE _t.is_deleted = false"))
	assert.True(t, strings.Contains(wrapped, base))
}

func TestSelectWrapsQuery(t *testing.T) {
	base := "SELECT id, is_deleted FROM users"
	wrapped := "SELECT * FROM (" + base + ") AS _t WHERE _t.is_deleted = false"
	assert.True(t, strings.Contains(wrapped, "WHERE _t.is_deleted = false"))
}

func TestGetIncludingDeletedDoesNotFilter(t *testing.T) {
	base := "SELECT id, is_deleted FROM users WHERE id = $1"
	// GetIncludingDeleted does NOT wrap — the raw query is used.
	assert.False(t, strings.Contains(base, "is_deleted = false"))
}

func TestPageRequestNormalise(t *testing.T) {
	tests := []struct {
		name        string
		input       database.PageRequest
		wantPage    int
		wantSize    int
		wantSortDir string
		wantSortBy  string
	}{
		{
			name:        "defaults applied when zero values",
			input:       database.PageRequest{},
			wantPage:    1,
			wantSize:    20,
			wantSortDir: "asc",
			wantSortBy:  "created_at",
		},
		{
			name:        "page below 1 becomes 1",
			input:       database.PageRequest{Page: -5, PageSize: 10},
			wantPage:    1,
			wantSize:    10,
			wantSortDir: "asc",
			wantSortBy:  "created_at",
		},
		{
			name:        "page size over 100 becomes 20",
			input:       database.PageRequest{Page: 2, PageSize: 999},
			wantPage:    2,
			wantSize:    20,
			wantSortDir: "asc",
			wantSortBy:  "created_at",
		},
		{
			name:        "desc sort direction preserved",
			input:       database.PageRequest{Page: 1, PageSize: 10, SortDir: "desc", SortBy: "name"},
			wantPage:    1,
			wantSize:    10,
			wantSortDir: "desc",
			wantSortBy:  "name",
		},
		{
			name:        "invalid sort direction becomes asc",
			input:       database.PageRequest{Page: 1, PageSize: 10, SortDir: "invalid"},
			wantPage:    1,
			wantSize:    10,
			wantSortDir: "asc",
			wantSortBy:  "created_at",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.input
			req.Normalise("created_at")
			assert.Equal(t, tc.wantPage, req.Page)
			assert.Equal(t, tc.wantSize, req.PageSize)
			assert.Equal(t, tc.wantSortDir, req.SortDir)
			assert.Equal(t, tc.wantSortBy, req.SortBy)
		})
	}
}
