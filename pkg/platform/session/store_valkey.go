package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	ddvalkey "github.com/DataDog/dd-trace-go/contrib/valkey-io/valkey-go/v2"
	"github.com/valkey-io/valkey-go"
)

type valkeyStore struct{ client valkey.Client }

func NewValkeyStore(client valkey.Client) SessionStore {
	return &valkeyStore{client: client}
}

func MustConnectValkey(url string) valkey.Client {
	client, err := ddvalkey.NewClient(valkey.ClientOption{InitAddress: []string{url}})
	if err != nil {
		panic(fmt.Sprintf("session: valkey connect: %v", err))
	}
	return client
}

func (s *valkeyStore) Create(ctx context.Context, userID string, ttl time.Duration, meta map[string]string) (*Session, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("session: generate id: %w", err)
	}
	sess := &Session{
		ID:        base64.RawURLEncoding.EncodeToString(b),
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(ttl),
		Metadata:  meta,
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return nil, fmt.Errorf("session: marshal: %w", err)
	}
	key := "session:" + sess.ID
	if err := s.client.Do(ctx, s.client.B().Set().Key(key).Value(string(data)).Ex(ttl).Build()).Error(); err != nil {
		return nil, fmt.Errorf("session: store: %w", err)
	}
	return sess, nil
}

func (s *valkeyStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := "session:" + sessionID
	val, err := s.client.Do(ctx, s.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		return nil, nil // not found → nil
	}
	var sess Session
	if err := json.Unmarshal([]byte(val), &sess); err != nil {
		return nil, fmt.Errorf("session: unmarshal: %w", err)
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, nil
	}
	return &sess, nil
}

func (s *valkeyStore) Destroy(ctx context.Context, sessionID string) error {
	return s.client.Do(ctx, s.client.B().Del().Key("session:"+sessionID).Build()).Error()
}

func (s *valkeyStore) Refresh(ctx context.Context, sessionID string, ttl time.Duration) error {
	sess, err := s.Get(ctx, sessionID)
	if err != nil || sess == nil {
		return fmt.Errorf("session: refresh: not found")
	}
	sess.ExpiresAt = time.Now().UTC().Add(ttl)
	data, _ := json.Marshal(sess)
	return s.client.Do(ctx,
		s.client.B().Set().Key("session:"+sessionID).Value(string(data)).Ex(ttl).Build()).Error()
}
