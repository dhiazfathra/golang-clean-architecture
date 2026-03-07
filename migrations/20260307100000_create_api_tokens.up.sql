CREATE TABLE IF NOT EXISTS api_tokens (
    id           BIGINT PRIMARY KEY,
    name         VARCHAR(100)  NOT NULL,
    token_hash   VARCHAR(64)   NOT NULL UNIQUE,
    token_prefix VARCHAR(12)   NOT NULL,
    user_id      VARCHAR(36)   NOT NULL,
    expires_at   TIMESTAMPTZ   NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    created_by   VARCHAR(36)   NOT NULL DEFAULT '',
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by   VARCHAR(36)   NOT NULL DEFAULT '',
    is_deleted   BOOLEAN       NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_api_tokens_user_id ON api_tokens (user_id) WHERE NOT is_deleted;
CREATE INDEX idx_api_tokens_hash    ON api_tokens (token_hash) WHERE NOT is_deleted;
