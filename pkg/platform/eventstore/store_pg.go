package eventstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type pgStore struct{ db *sqlx.DB }

func NewPgStore(db *sqlx.DB) EventStore { return &pgStore{db: db} }

func (s *pgStore) Append(ctx context.Context, events []Event) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("eventstore: begin tx: %w", err)
	}
	defer tx.Rollback()
	for _, e := range events {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("eventstore: marshal event: %w", err)
		}
		meta, err := json.Marshal(e.Metadata())
		if err != nil {
			return fmt.Errorf("eventstore: marshal metadata: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO events (aggregate_type, aggregate_id, event_type, version, data, metadata, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			e.AggregateType(), e.AggregateID(), e.EventType(), e.Version(),
			data, meta, e.Timestamp())
		if err != nil {
			return fmt.Errorf("eventstore: append: %w", err)
		}
	}
	return tx.Commit()
}

func (s *pgStore) Load(ctx context.Context, aggregateType, aggregateID string, fromVersion int) ([]Event, error) {
	rows, err := s.db.QueryxContext(ctx, `
		SELECT event_type, data, metadata FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND version > $3
		ORDER BY version ASC`,
		aggregateType, aggregateID, fromVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("eventstore: load: %w", err)
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var evType string
		var data, meta []byte
		if err := rows.Scan(&evType, &data, &meta); err != nil {
			return nil, err
		}
		e, err := Deserialise(evType, data)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
