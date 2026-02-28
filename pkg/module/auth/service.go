package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

const sessionTTL = 24 * time.Hour

type Service struct {
	sessions session.SessionStore
	users    UserProvider
	hasher   PasswordHasher
}

func NewService(sessions session.SessionStore, users UserProvider, hasher PasswordHasher) *Service {
	return &Service{sessions: sessions, users: users, hasher: hasher}
}

func (s *Service) Login(ctx context.Context, req LoginRequest, meta map[string]string) (*session.Session, error) {
	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if !user.Active {
		return nil, fmt.Errorf("account disabled")
	}
	if !s.hasher.Verify(req.Password, user.PassHash) {
		return nil, fmt.Errorf("invalid credentials")
	}
	return s.sessions.Create(ctx, user.ID, sessionTTL, meta)
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.sessions.Destroy(ctx, sessionID)
}

func (s *Service) CurrentUser(ctx context.Context, userID string) (*UserRecord, error) {
	return s.users.GetByEmail(ctx, userID) // UserProvider may expose GetByID separately
}
