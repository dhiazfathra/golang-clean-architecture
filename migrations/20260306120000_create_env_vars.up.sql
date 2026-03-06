CREATE TABLE IF NOT EXISTS env_vars (
    id          BIGINT PRIMARY KEY,
    platform    VARCHAR(30) NOT NULL,
    key         VARCHAR(50) NOT NULL,
    value       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  VARCHAR(255) NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by  VARCHAR(255) NOT NULL DEFAULT '',
    is_deleted  BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (platform, key)
);

CREATE INDEX idx_env_vars_platform_key ON env_vars (platform, key) WHERE NOT is_deleted;
CREATE INDEX idx_env_vars_platform ON env_vars (platform) WHERE NOT is_deleted;
