package auth

import "fmt"

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
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
	GetByEmail(ctx interface{ Value(any) any }, email string) (*UserRecord, error)
}

type UserRecord struct {
	ID       string
	Email    string
	PassHash string
	Active   bool
}
