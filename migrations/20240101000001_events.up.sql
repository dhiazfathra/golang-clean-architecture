-- This migration is intentionally irreversible (no down migration).
CREATE TABLE events (
    id              BIGSERIAL    PRIMARY KEY,
    aggregate_type  TEXT         NOT NULL,
    aggregate_id    TEXT         NOT NULL,
    event_type      TEXT         NOT NULL,
    version         INT          NOT NULL,
    data            JSONB        NOT NULL,
    metadata        JSONB        NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (aggregate_type, aggregate_id, version)
);
CREATE INDEX idx_events_aggregate ON events (aggregate_type, aggregate_id, version);
CREATE INDEX idx_events_created   ON events (created_at);
