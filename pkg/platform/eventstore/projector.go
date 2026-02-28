package eventstore

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
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
}

func NewProjectionRunner(db *sqlx.DB, store EventStore) *ProjectionRunner {
	return &ProjectionRunner{db: db, store: store, interval: 500 * time.Millisecond}
}

func (r *ProjectionRunner) Register(p Projector) { r.projectors = append(r.projectors, p) }

func (r *ProjectionRunner) Start(ctx context.Context) {
	for _, p := range r.projectors {
		go r.runProjector(ctx, p)
	}
}

func (r *ProjectionRunner) runProjector(ctx context.Context, p Projector) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.poll(ctx, p); err != nil {
				slog.Error("projection poll error", "projector", p.Name(), "error", err)
			}
		}
	}
}

func (r *ProjectionRunner) poll(ctx context.Context, p Projector) error {
	var lastID int64
	err := r.db.GetContext(ctx, &lastID,
		`SELECT last_event_id FROM projection_cursors WHERE projector_name = $1`, p.Name())
	if err != nil {
		// First run — insert cursor at 0
		_, err = r.db.ExecContext(ctx,
			`INSERT INTO projection_cursors (projector_name, last_event_id) VALUES ($1, 0) ON CONFLICT DO NOTHING`, p.Name())
		if err != nil {
			return fmt.Errorf("init cursor: %w", err)
		}
		lastID = 0
	}
	rows, err := r.db.QueryxContext(ctx, `
		SELECT id, event_type, data, metadata FROM events WHERE id > $1 ORDER BY id ASC LIMIT 100`, lastID)
	if err != nil {
		return fmt.Errorf("poll events: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var evType string
		var data, meta []byte
		if err := rows.Scan(&id, &evType, &data, &meta); err != nil {
			return err
		}
		e, err := Deserialise(evType, data)
		if err != nil {
			slog.Warn("unknown event type, skipping", "type", evType, "id", id)
			lastID = id
			continue
		}
		if err := p.Handle(ctx, e); err != nil {
			return fmt.Errorf("handle event %d: %w", id, err)
		}
		lastID = id
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE projection_cursors SET last_event_id = $1 WHERE projector_name = $2`, lastID, p.Name())
	return err
}
