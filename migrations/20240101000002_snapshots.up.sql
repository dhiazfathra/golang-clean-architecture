CREATE TABLE snapshots (
    aggregate_type  TEXT         NOT NULL,
    aggregate_id    TEXT         NOT NULL,
    version         INT          NOT NULL,
    data            JSONB        NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (aggregate_type, aggregate_id)
);
