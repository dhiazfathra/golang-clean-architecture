package main

import (
	"context"
	"errors"
	"testing"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
)

const unexpectedErrorFmt = "unexpected error: %v"
const expectedError = "expected an error"

func TestOrderUserProvider_GetByID_Found(t *testing.T) {
	t.Parallel()
	svc := &user.MockUserService{GetByIDResult: &user.UserReadModel{}}
	oup := &orderUserProvider{svc: svc}

	exists, err := oup.GetByID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if !exists {
		t.Error("expected exists=true when service returns a non-nil user")
	}
}

func TestOrderUserProvider_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	svc := &user.MockUserService{GetByIDResult: nil}
	oup := &orderUserProvider{svc: svc}

	exists, err := oup.GetByID(context.Background(), "missing")
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if exists {
		t.Error("expected exists=false when service returns nil")
	}
}

func TestOrderUserProvider_GetByID_Error(t *testing.T) {
	t.Parallel()
	svc := &user.MockUserService{GetByIDErr: errors.New("db error")}
	oup := &orderUserProvider{svc: svc}

	_, err := oup.GetByID(context.Background(), "user-1")
	if err == nil {
		t.Error(expectedError)
	}
}

// --- authUserAdapter ---

func TestAuthUserAdapter_GetByEmail_Success(t *testing.T) {
	t.Parallel()
	want := &auth.UserRecord{Email: "a@b.com"}
	svc := &user.MockUserService{GetByEmailAuthResult: want}
	aua := &authUserAdapter{svc: svc}

	got, err := aua.GetByEmail(context.Background(), "a@b.com")
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAuthUserAdapter_GetByEmail_Error(t *testing.T) {
	t.Parallel()
	svc := &user.MockUserService{GetByEmailAuthErr: errors.New("not found")}
	aua := &authUserAdapter{svc: svc}

	_, err := aua.GetByEmail(context.Background(), "x@y.com")
	if err == nil {
		t.Error(expectedError)
	}
}

// --- seederUserAdapter ---

func TestSeederUserAdapter_GetByEmail_Success(t *testing.T) {
	t.Parallel()
	want := &seeder.UserRecord{Email: "seed@test.com"}
	svc := &user.MockUserService{GetByEmailSeederResult: want}
	sua := &seederUserAdapter{svc: svc}

	got, err := sua.GetByEmail(context.Background(), "seed@test.com")
	if err != nil {
		t.Fatalf(unexpectedErrorFmt, err)
	}
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

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
		t.Fatalf(unexpectedErrorFmt, err)
	}
}

// --- Interface compliance (compile-time) ---

var _ interface {
	GetByID(ctx context.Context, id string) (bool, error)
} = (*orderUserProvider)(nil)

var _ interface {
	GetByEmail(ctx context.Context, email string) (*auth.UserRecord, error)
} = (*authUserAdapter)(nil)

var _ interface {
	CreateUser(ctx context.Context, cmd seeder.CreateUserCmd) (string, error)
	GetByEmail(ctx context.Context, email string) (*seeder.UserRecord, error)
	AssignRole(ctx context.Context, userID, roleID, actor string) error
} = (*seederUserAdapter)(nil)
