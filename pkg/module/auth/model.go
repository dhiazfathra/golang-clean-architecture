package auth

import (
	"context"
	"fmt"
)

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"` //nolint:gosec // G117: false positive, this is a request DTO not a hardcoded secret
}

func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return fmt.Errorf("email required")
	}
	if r.Password == "" {
		return fmt.Errorf("password required")
	}
	return nil
}

// UserProvider is satisfied by user.Service — avoids import cycle.
type UserProvider interface {
	GetByEmail(ctx context.Context, email string) (*UserRecord, error)
}

type UserRecord struct {
	ID       string
	Email    string
	PassHash string
	Active   bool
}
