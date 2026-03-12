package eventstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

type Projector interface {
	Name() string
	Handle(ctx context.Context, event Event) error
}

type ProjectionRunner struct {
	db         *sqlx.DB
	store      EventStore
	projectors []Projector
	interval   time.Duration
	logger     zerolog.Logger
}

func NewProjectionRunner(db *sqlx.DB, store EventStore, logger zerolog.Logger) *ProjectionRunner {
	return &ProjectionRunner{db: db, store: store, interval: 500 * time.Millisecond, logger: logger}
}

func (r *ProjectionRunner) Register(p Projector) { r.projectors = append(r.projectors, p) }

func (r *ProjectionRunner) Start(ctx context.Context) {
	for _, p := range r.projectors {
		go r.runProjector(ctx, p)
	}
}

// RunOnce synchronously drains all pending events for every registered projector.
// Use this to flush pending events before reading from projections.
func (r *ProjectionRunner) RunOnce(ctx context.Context) error {
	for _, p := range r.projectors {
		for {
			n, err := r.poll(ctx, p)
			if err != nil {
				return fmt.Errorf("run once %s: %w", p.Name(), err)
			}
			if n == 0 {
				break
			}
		}
	}
	return nil
}

func (r *ProjectionRunner) runProjector(ctx context.Context, p Projector) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := r.poll(ctx, p); err != nil {
				r.logger.Error().Err(err).Str("projector", p.Name()).Msg("projection poll error")
			}
		}
	}
}

func (r *ProjectionRunner) poll(ctx context.Context, p Projector) (int, error) {
	var lastID int64
	err := r.db.GetContext(ctx, &lastID,
		`SELECT last_event_id FROM projection_cursors WHERE projector_name = $1`, p.Name())
	if err != nil {
		// First run — insert cursor at 0
		_, err = r.db.ExecContext(ctx,
			`INSERT INTO projection_cursors (projector_name, last_event_id) VALUES ($1, 0) ON CONFLICT DO NOTHING`, p.Name())
		if err != nil {
			return 0, fmt.Errorf("init cursor: %w", err)
		}
		lastID = 0
	}
	rows, err := r.db.QueryxContext(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, version, data, metadata, created_at
		FROM events WHERE id > $1 ORDER BY id ASC LIMIT 100`, lastID)
	if err != nil {
		return 0, fmt.Errorf("poll events: %w", err)
	}
	defer rows.Close()
	var n int
	for rows.Next() {
		var id int64
		var aggType, aggID, evType string
		var version int
		var data, meta []byte
		var createdAt time.Time
		if err := rows.Scan(&id, &aggType, &aggID, &evType, &version, &data, &meta, &createdAt); err != nil {
			return n, err
		}
		e, err := Deserialise(evType, data)
		if err != nil {
			r.logger.Warn().Str("type", evType).Int64("id", id).Msg("unknown event type, skipping")
			lastID = id
			n++
			continue
		}
		if env, ok := e.(Enveloper); ok {
			var metaMap map[string]string
			_ = json.Unmarshal(meta, &metaMap)
			env.SetEnvelope(aggID, aggType, evType, version, createdAt, metaMap)
		}
		if err := p.Handle(ctx, e); err != nil {
			return n, fmt.Errorf("handle event %d: %w", id, err)
		}
		lastID = id
		n++
	}
	if err := rows.Err(); err != nil {
		return n, err
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE projection_cursors SET last_event_id = $1 WHERE projector_name = $2`, lastID, p.Name())
	return n, err
}
