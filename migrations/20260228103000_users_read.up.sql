CREATE TABLE users_read (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    pass_hash   TEXT NOT NULL DEFAULT '',
    active      BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL,
    created_by  TEXT        NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    updated_by  TEXT        NOT NULL,
    is_deleted  BOOLEAN     NOT NULL DEFAULT false
);
CREATE INDEX idx_users_read_email       ON users_read (email) WHERE is_deleted = false;
CREATE INDEX idx_users_read_not_deleted ON users_read (id)    WHERE is_deleted = false;
