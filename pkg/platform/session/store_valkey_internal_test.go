package session

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

type errReader struct{}

func (errReader) Read(p []byte) (int, error) {
	return 0, errors.New("rand fail")
}

func TestValkeyStoreCreateRandReadError(t *testing.T) {
	url := testutil.SetupTestValkey(t)
	store := NewValkeyStore(url)

	origRandRead := randRead
	defer func() { randRead = origRandRead }()
	randRead = errReader{}.Read

	got, err := store.Create(context.Background(), "user", time.Minute, nil)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "session: generate id")
}

func TestValkeyStoreCreateMarshalError(t *testing.T) {
	client := testutil.SetupTestValkey(t)
	store := NewValkeyStore(client)

	origMarshal := jsonMarshal
	defer func() { jsonMarshal = origMarshal }()
	jsonMarshal = func(any) ([]byte, error) {
		return nil, errors.New("marshal fail")
	}

	got, err := store.Create(context.Background(), "user", time.Minute, nil)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "session: marshal")
}

func TestValkeyStoreGetExpiredSessionReturnsNil(t *testing.T) {
	client := testutil.SetupTestValkey(t)
	store := NewValkeyStore(client)
	ctx := context.Background()

	payload, err := json.Marshal(&Session{
		ID:        "expired-session",
		UserID:    "user-expired",
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-time.Minute),
	})
	require.NoError(t, err)
	err = client.Do(ctx, client.B().Set().Key("session:expired-session").Value(string(payload)).Ex(time.Minute).Build()).Error()
	require.NoError(t, err)

	got, err := store.Get(ctx, "expired-session")
	require.NoError(t, err)
	assert.Nil(t, got)
}
