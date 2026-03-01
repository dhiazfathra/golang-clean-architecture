CREATE TABLE IF NOT EXISTS feature_flags (
    id          BIGINT PRIMARY KEY,
    key         VARCHAR(255) NOT NULL UNIQUE,
    enabled     BOOLEAN NOT NULL DEFAULT false,
    description TEXT NOT NULL DEFAULT '',
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  VARCHAR(255) NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by  VARCHAR(255) NOT NULL DEFAULT '',
    is_deleted  BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_feature_flags_key ON feature_flags (key) WHERE NOT is_deleted;
