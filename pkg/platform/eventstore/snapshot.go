package eventstore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

const defaultSnapshotFrequency = 100

type SnapshotStore struct {
	db        *sqlx.DB
	frequency int
}

func NewSnapshotStore(db *sqlx.DB) *SnapshotStore {
	return &SnapshotStore{db: db, frequency: defaultSnapshotFrequency}
}

// SnapshotSave persists a snapshot when the aggregate version is a multiple of
// the store's frequency. Methods cannot have type parameters in Go, so this is
// a package-level generic function.
func SnapshotSave[T any](ctx context.Context, s *SnapshotStore, agg *Aggregate[T]) error {
	if agg.Version%s.frequency != 0 {
		return nil
	}
	data, err := json.Marshal(agg.State)
	if err != nil {
		return fmt.Errorf("snapshot: marshal: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO snapshots (aggregate_type, aggregate_id, version, data, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (aggregate_type, aggregate_id) DO UPDATE SET version=$3, data=$4, created_at=NOW()`,
		agg.ID, agg.ID, agg.Version, data)
	return err
}

// SnapshotLoad restores an aggregate from the latest snapshot, or returns nil
// if no snapshot exists.
func SnapshotLoad[T any](ctx context.Context, s *SnapshotStore, aggregateType, aggregateID string) (*Aggregate[T], error) {
	var version int
	var data []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT version, data FROM snapshots WHERE aggregate_type=$1 AND aggregate_id=$2`,
		aggregateType, aggregateID).Scan(&version, &data)
	if err != nil {
		return nil, nil // no snapshot
	}
	var state T
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("snapshot: unmarshal: %w", err)
	}
	return &Aggregate[T]{ID: aggregateID, Version: version, State: state}, nil
}
