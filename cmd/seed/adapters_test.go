package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/order"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/apitoken"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/envvar"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/featureflag"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
)

const unexpectedErrorFmt = "unexpected error: %v"
const expectedError = "expected an error"

// ---------------------------------------------------------------------------
// Mock: ffService
// ---------------------------------------------------------------------------

type mockFFService struct {
	createResult *featureflag.Flag
	createErr    error
	listResult   []featureflag.Flag
	listErr      error
}

func (m *mockFFService) Create(_ context.Context, _, _ string, _ bool, _ string) (*featureflag.Flag, error) {
	return m.createResult, m.createErr
}
func (m *mockFFService) List(_ context.Context) ([]featureflag.Flag, error) {
	return m.listResult, m.listErr
}

// ---------------------------------------------------------------------------
// Mock: evService
// ---------------------------------------------------------------------------

type mockEVService struct {
	createResult     *envvar.EnvVar
	createErr        error
	listByPlatResult *database.PageResponse[envvar.EnvVar]
	listByPlatErr    error
}

func (m *mockEVService) Create(_ context.Context, _, _, _, _ string) (*envvar.EnvVar, error) {
	return m.createResult, m.createErr
}
func (m *mockEVService) ListByPlatform(_ context.Context, _ string, _ database.PageRequest) (*database.PageResponse[envvar.EnvVar], error) {
	return m.listByPlatResult, m.listByPlatErr
}

// ---------------------------------------------------------------------------
// Mock: atService
// ---------------------------------------------------------------------------

type mockATService struct {
	createRaw   string
	createToken *apitoken.APIToken
	createErr   error
	listResult  []apitoken.APIToken
	listErr     error
}

func (m *mockATService) Create(_ context.Context, _, _ string, _ time.Duration) (string, *apitoken.APIToken, error) {
	return m.createRaw, m.createToken, m.createErr
}
func (m *mockATService) List(_ context.Context, _ string) ([]apitoken.APIToken, error) {
	return m.listResult, m.listErr
}

// ---------------------------------------------------------------------------
// Mock: orderSvc
// ---------------------------------------------------------------------------

type mockOrderSvc struct {
	createOrderResult string
	createOrderErr    error
	listResult        *order.ListResponse
	listErr           error
}

func (m *mockOrderSvc) CreateOrder(_ context.Context, _ order.CreateOrderCmd) (string, error) {
	return m.createOrderResult, m.createOrderErr
}
func (m *mockOrderSvc) List(_ context.Context, _ order.ListRequest) (*order.ListResponse, error) {
	return m.listResult, m.listErr
}

// ---------------------------------------------------------------------------
// seederUserAdapter
// ---------------------------------------------------------------------------

func TestSeederUserAdapter_CreateUser_Success(t *testing.T) {
	t.Parallel()
	svc := &user.MockUserService{CreateUserResult: "new-id"}
	sua := &seederUserAdapter{svc: svc}

	id, err := sua.CreateUser(context.Background(), seeder.CreateUserCmd{})
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if id != "new-id" {
		t.Errorf("got id=%q, want %q", id, "new-id")
	}
}

func TestSeederUserAdapter_CreateUser_Error(t *testing.T) {
	t.Parallel()
	svc := &user.MockUserService{CreateUserErr: errors.New("create failed")}
	sua := &seederUserAdapter{svc: svc}

	_, err := sua.CreateUser(context.Background(), seeder.CreateUserCmd{})
	if err == nil {
		t.Error(expectedError)
	}
}

func TestSeederUserAdapter_GetByEmail_Success(t *testing.T) {
	t.Parallel()
	want := &seeder.UserRecord{Email: "seed@test.com"}
	svc := &user.MockUserService{GetByEmailSeederResult: want}
	sua := &seederUserAdapter{svc: svc}

	got, err := sua.GetByEmail(context.Background(), "seed@test.com")
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if got.Email != want.Email {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSeederUserAdapter_GetByEmail_Error(t *testing.T) {
	t.Parallel()
	svc := &user.MockUserService{GetByEmailSeederErr: errors.New("not found")}
	sua := &seederUserAdapter{svc: svc}

	_, err := sua.GetByEmail(context.Background(), "missing@test.com")
	if err == nil {
		t.Error(expectedError)
	}
}

func TestSeederUserAdapter_AssignRole_Success(t *testing.T) {
	t.Parallel()
	svc := &user.MockUserService{}
	sua := &seederUserAdapter{svc: svc}

	if err := sua.AssignRole(context.Background(), "u1", "r1", "actor"); err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
}

func TestSeederUserAdapter_AssignRole_Error(t *testing.T) {
	t.Parallel()
	svc := &user.MockUserService{AssignRoleErr: errors.New("forbidden")}
	sua := &seederUserAdapter{svc: svc}

	if err := sua.AssignRole(context.Background(), "u1", "r1", "actor"); err == nil {
		t.Error(expectedError)
	}
}

// ---------------------------------------------------------------------------
// seederFFAdapter
// ---------------------------------------------------------------------------

func TestSeederFFAdapter_Create_Success(t *testing.T) {
	t.Parallel()
	mock := &mockFFService{
		createResult: &featureflag.Flag{ID: 1, Key: "new_ui", Enabled: true, Description: "desc"},
	}
	a := &seederFFAdapter{svc: mock}

	got, err := a.Create(context.Background(), "new_ui", "desc", true, "user-1")
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if got.Key != "new_ui" {
		t.Errorf("got key=%q, want %q", got.Key, "new_ui")
	}
	if got.ID != 1 {
		t.Errorf("got ID=%d, want 1", got.ID)
	}
	if !got.Enabled {
		t.Error("expected Enabled=true")
	}
	if got.Description != "desc" {
		t.Errorf("got Description=%q, want %q", got.Description, "desc")
	}
}

func TestSeederFFAdapter_Create_Error(t *testing.T) {
	t.Parallel()
	mock := &mockFFService{createErr: errors.New("db error")}
	a := &seederFFAdapter{svc: mock}

	_, err := a.Create(context.Background(), "k", "d", false, "u")
	if err == nil {
		t.Error(expectedError)
	}
}

func TestSeederFFAdapter_List_Success(t *testing.T) {
	t.Parallel()
	mock := &mockFFService{
		listResult: []featureflag.Flag{
			{ID: 1, Key: "k1", Enabled: true, Description: "d1"},
			{ID: 2, Key: "k2", Enabled: false, Description: "d2"},
		},
	}
	a := &seederFFAdapter{svc: mock}

	flags, err := a.List(context.Background())
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if len(flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(flags))
	}
	if flags[0].Key != "k1" || flags[1].Key != "k2" {
		t.Errorf("unexpected flag keys: %v", flags)
	}
}

func TestSeederFFAdapter_List_Empty(t *testing.T) {
	t.Parallel()
	mock := &mockFFService{listResult: []featureflag.Flag{}}
	a := &seederFFAdapter{svc: mock}

	flags, err := a.List(context.Background())
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if len(flags) != 0 {
		t.Errorf("expected empty list, got %d items", len(flags))
	}
}

func TestSeederFFAdapter_List_Error(t *testing.T) {
	t.Parallel()
	mock := &mockFFService{listErr: errors.New("db error")}
	a := &seederFFAdapter{svc: mock}

	_, err := a.List(context.Background())
	if err == nil {
		t.Error(expectedError)
	}
}

// ---------------------------------------------------------------------------
// seederEnvVarAdapter
// ---------------------------------------------------------------------------

func TestSeederEnvVarAdapter_Create_Success(t *testing.T) {
	t.Parallel()
	mock := &mockEVService{
		createResult: &envvar.EnvVar{ID: 10, Platform: "web", Key: "APP_NAME", Value: "myapp"},
	}
	a := &seederEnvVarAdapter{svc: mock}

	got, err := a.Create(context.Background(), "web", "APP_NAME", "myapp", "system")
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if got.Platform != "web" || got.Key != "APP_NAME" || got.Value != "myapp" || got.ID != 10 {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestSeederEnvVarAdapter_Create_Error(t *testing.T) {
	t.Parallel()
	mock := &mockEVService{createErr: errors.New("insert error")}
	a := &seederEnvVarAdapter{svc: mock}

	_, err := a.Create(context.Background(), "web", "KEY", "val", "system")
	if err == nil {
		t.Error(expectedError)
	}
}

func TestSeederEnvVarAdapter_ListByPlatform_Success(t *testing.T) {
	t.Parallel()
	mock := &mockEVService{
		listByPlatResult: &database.PageResponse[envvar.EnvVar]{
			Items: []envvar.EnvVar{
				{ID: 1, Platform: "web", Key: "K1", Value: "V1"},
				{ID: 2, Platform: "web", Key: "K2", Value: "V2"},
			},
			Total:      2,
			Page:       1,
			PageSize:   20,
			TotalPages: 1,
		},
	}
	a := &seederEnvVarAdapter{svc: mock}

	resp, err := a.ListByPlatform(context.Background(), "web", database.PageRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].Key != "K1" || resp.Items[1].Key != "K2" {
		t.Errorf("unexpected keys: %v", resp.Items)
	}
	if resp.Total != 2 || resp.Page != 1 || resp.PageSize != 20 || resp.TotalPages != 1 {
		t.Errorf("unexpected pagination: %+v", resp)
	}
}

func TestSeederEnvVarAdapter_ListByPlatform_Error(t *testing.T) {
	t.Parallel()
	mock := &mockEVService{listByPlatErr: errors.New("query error")}
	a := &seederEnvVarAdapter{svc: mock}

	_, err := a.ListByPlatform(context.Background(), "web", database.PageRequest{})
	if err == nil {
		t.Error(expectedError)
	}
}

// ---------------------------------------------------------------------------
// seederAPITokenAdapter
// ---------------------------------------------------------------------------

func TestSeederAPITokenAdapter_Create_Success(t *testing.T) {
	t.Parallel()
	exp := time.Now().Add(24 * time.Hour)
	mock := &mockATService{
		createRaw: "raw-secret-token",
		createToken: &apitoken.APIToken{
			ID:          99,
			Name:        "dev_token",
			TokenHash:   "hash123",
			TokenPrefix: "dev_",
			UserID:      "user-1",
			ExpiresAt:   exp,
		},
	}
	a := &seederAPITokenAdapter{svc: mock}

	raw, token, err := a.Create(context.Background(), "dev_token", "user-1", 24*time.Hour)
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if raw != "raw-secret-token" {
		t.Errorf("got raw=%q, want %q", raw, "raw-secret-token")
	}
	if token.ID != 99 || token.Name != "dev_token" || token.TokenHash != "hash123" {
		t.Errorf("unexpected token: %+v", token)
	}
	if token.TokenPrefix != "dev_" || token.UserID != "user-1" {
		t.Errorf("unexpected token fields: %+v", token)
	}
	if !token.ExpiresAt.Equal(exp) {
		t.Errorf("ExpiresAt mismatch: got %v, want %v", token.ExpiresAt, exp)
	}
}

func TestSeederAPITokenAdapter_Create_Error(t *testing.T) {
	t.Parallel()
	mock := &mockATService{createErr: errors.New("token error")}
	a := &seederAPITokenAdapter{svc: mock}

	_, _, err := a.Create(context.Background(), "name", "uid", time.Hour)
	if err == nil {
		t.Error(expectedError)
	}
}

func TestSeederAPITokenAdapter_List_Success(t *testing.T) {
	t.Parallel()
	exp := time.Now().Add(time.Hour)
	mock := &mockATService{
		listResult: []apitoken.APIToken{
			{ID: 1, Name: "t1", TokenHash: "h1", TokenPrefix: "p1", UserID: "u1", ExpiresAt: exp},
			{ID: 2, Name: "t2", TokenHash: "h2", TokenPrefix: "p2", UserID: "u1", ExpiresAt: exp},
		},
	}
	a := &seederAPITokenAdapter{svc: mock}

	tokens, err := a.List(context.Background(), "u1")
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Name != "t1" || tokens[1].Name != "t2" {
		t.Errorf("unexpected token names: %v", tokens)
	}
	if tokens[0].TokenHash != "h1" || tokens[0].TokenPrefix != "p1" {
		t.Errorf("unexpected token[0] fields: %+v", tokens[0])
	}
}

func TestSeederAPITokenAdapter_List_Empty(t *testing.T) {
	t.Parallel()
	mock := &mockATService{listResult: []apitoken.APIToken{}}
	a := &seederAPITokenAdapter{svc: mock}

	tokens, err := a.List(context.Background(), "u1")
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected empty list, got %d", len(tokens))
	}
}

func TestSeederAPITokenAdapter_List_Error(t *testing.T) {
	t.Parallel()
	mock := &mockATService{listErr: errors.New("list error")}
	a := &seederAPITokenAdapter{svc: mock}

	_, err := a.List(context.Background(), "u1")
	if err == nil {
		t.Error(expectedError)
	}
}

// ---------------------------------------------------------------------------
// seederOrderAdapter
// ---------------------------------------------------------------------------

func TestSeederOrderAdapter_CreateOrder_Success(t *testing.T) {
	t.Parallel()
	mock := &mockOrderSvc{createOrderResult: "order-123"}
	a := &seederOrderAdapter{svc: mock}

	id, err := a.CreateOrder(context.Background(), seeder.CreateOrderCmd{
		UserID: "user-1",
		Total:  99.99,
		Actor:  "system",
	})
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if id != "order-123" {
		t.Errorf("got id=%q, want %q", id, "order-123")
	}
}

func TestSeederOrderAdapter_CreateOrder_Error(t *testing.T) {
	t.Parallel()
	mock := &mockOrderSvc{createOrderErr: errors.New("order error")}
	a := &seederOrderAdapter{svc: mock}

	_, err := a.CreateOrder(context.Background(), seeder.CreateOrderCmd{})
	if err == nil {
		t.Error(expectedError)
	}
}

func TestSeederOrderAdapter_List_Success(t *testing.T) {
	t.Parallel()
	mock := &mockOrderSvc{
		listResult: &order.ListResponse{
			Items: []order.OrderReadModel{
				{ID: 10, UserID: 20, Total: 50.0},
				{ID: 11, UserID: 21, Total: 75.5},
			},
			Total: 2,
		},
	}
	a := &seederOrderAdapter{svc: mock}

	resp, err := a.List(context.Background(), seeder.ListRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].ID != "10" || resp.Items[0].UserID != "20" {
		t.Errorf("unexpected item[0]: %+v", resp.Items[0])
	}
	if resp.Items[1].ID != "11" || resp.Items[1].UserID != "21" {
		t.Errorf("unexpected item[1]: %+v", resp.Items[1])
	}
	if resp.Items[0].Total != 50.0 || resp.Items[1].Total != 75.5 {
		t.Errorf("unexpected totals: %v", resp.Items)
	}
	if resp.Total != 2 {
		t.Errorf("got Total=%d, want 2", resp.Total)
	}
}

func TestSeederOrderAdapter_List_Empty(t *testing.T) {
	t.Parallel()
	mock := &mockOrderSvc{
		listResult: &order.ListResponse{Items: []order.OrderReadModel{}, Total: 0},
	}
	a := &seederOrderAdapter{svc: mock}

	resp, err := a.List(context.Background(), seeder.ListRequest{})
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(resp.Items))
	}
}

func TestSeederOrderAdapter_List_Error(t *testing.T) {
	t.Parallel()
	mock := &mockOrderSvc{listErr: errors.New("list error")}
	a := &seederOrderAdapter{svc: mock}

	_, err := a.List(context.Background(), seeder.ListRequest{})
	if err == nil {
		t.Error(expectedError)
	}
}

// ---------------------------------------------------------------------------
// Compile-time interface compliance checks
// ---------------------------------------------------------------------------

var _ ffService = (*mockFFService)(nil)
var _ evService = (*mockEVService)(nil)
var _ atService = (*mockATService)(nil)
var _ orderSvc = (*mockOrderSvc)(nil)

var _ seeder.FeatureFlagCreator = (*seederFFAdapter)(nil)
var _ seeder.EnvVarCreator = (*seederEnvVarAdapter)(nil)
var _ seeder.APITokenCreator = (*seederAPITokenAdapter)(nil)
var _ seeder.OrderCreator = (*seederOrderAdapter)(nil)
