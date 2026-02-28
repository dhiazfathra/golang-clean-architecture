-- M5.1: Migrate all entity primary keys from TEXT to BIGINT (Snowflake int64).
-- Read-model tables contain only projectable data; existing rows are re-seeded on boot.

DROP TABLE IF EXISTS user_roles_read;
DROP TABLE IF EXISTS users_read;
DROP TABLE IF EXISTS permissions_read;
DROP TABLE IF EXISTS roles_read;

CREATE TABLE roles_read (
    id          BIGINT      PRIMARY KEY,
    name        TEXT        NOT NULL UNIQUE,
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL,
    created_by  TEXT        NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    updated_by  TEXT        NOT NULL,
    is_deleted  BOOLEAN     NOT NULL DEFAULT false
);
CREATE INDEX idx_roles_read_not_deleted ON roles_read (id) WHERE is_deleted = false;

CREATE TABLE permissions_read (
    id          BIGINT      PRIMARY KEY,
    role_id     BIGINT      NOT NULL,
    module      TEXT        NOT NULL,
    action      TEXT        NOT NULL,
    field_mode  TEXT        NOT NULL DEFAULT 'all',
    field_list  TEXT[]      NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL,
    created_by  TEXT        NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    updated_by  TEXT        NOT NULL,
    is_deleted  BOOLEAN     NOT NULL DEFAULT false,
    UNIQUE (role_id, module, action)
);
CREATE INDEX idx_permissions_role   ON permissions_read (role_id) WHERE is_deleted = false;
CREATE INDEX idx_permissions_lookup ON permissions_read (module, action) WHERE is_deleted = false;

CREATE TABLE user_roles_read (
    user_id     BIGINT      NOT NULL,
    role_id     BIGINT      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    created_by  TEXT        NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    updated_by  TEXT        NOT NULL,
    is_deleted  BOOLEAN     NOT NULL DEFAULT false,
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX idx_user_roles_not_deleted ON user_roles_read (user_id) WHERE is_deleted = false;

CREATE TABLE users_read (
    id          BIGINT      PRIMARY KEY,
    email       TEXT        NOT NULL UNIQUE,
    pass_hash   TEXT        NOT NULL DEFAULT '',
    active      BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL,
    created_by  TEXT        NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    updated_by  TEXT        NOT NULL,
    is_deleted  BOOLEAN     NOT NULL DEFAULT false
);
CREATE INDEX idx_users_read_email       ON users_read (email) WHERE is_deleted = false;
CREATE INDEX idx_users_read_not_deleted ON users_read (id)    WHERE is_deleted = false;
