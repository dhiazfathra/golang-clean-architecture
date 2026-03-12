-- This migration is intentionally irreversible (no down migration).
CREATE TABLE projection_cursors (
    projector_name  TEXT    PRIMARY KEY,
    last_event_id   BIGINT  NOT NULL DEFAULT 0
);
