CREATE TABLE user_roles_read (
    user_id     TEXT NOT NULL,
    role_id     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    created_by  TEXT        NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    updated_by  TEXT        NOT NULL,
    is_deleted  BOOLEAN     NOT NULL DEFAULT false,
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX idx_user_roles_not_deleted ON user_roles_read (user_id) WHERE is_deleted = false;
