CREATE TABLE permissions_read (
    id          TEXT PRIMARY KEY,
    role_id     TEXT NOT NULL,
    module      TEXT NOT NULL,
    action      TEXT NOT NULL,
    field_mode  TEXT NOT NULL DEFAULT 'all',
    field_list  TEXT[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL,
    created_by  TEXT        NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    updated_by  TEXT        NOT NULL,
    is_deleted  BOOLEAN     NOT NULL DEFAULT false
);
CREATE INDEX idx_permissions_role   ON permissions_read (role_id) WHERE is_deleted = false;
CREATE INDEX idx_permissions_lookup ON permissions_read (module, action) WHERE is_deleted = false;
