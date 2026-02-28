package session

import (
	"context"
	"time"
)

type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	Metadata  map[string]string
}

type SessionStore interface {
	Create(ctx context.Context, userID string, ttl time.Duration, meta map[string]string) (*Session, error)
	Get(ctx context.Context, sessionID string) (*Session, error)
	Destroy(ctx context.Context, sessionID string) error
	Refresh(ctx context.Context, sessionID string, ttl time.Duration) error
}
