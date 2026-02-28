CREATE TABLE orders_read (
    id          BIGINT      PRIMARY KEY,
    user_id     BIGINT      NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'pending',
    total       DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL,
    created_by  TEXT        NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    updated_by  TEXT        NOT NULL,
    is_deleted  BOOLEAN     NOT NULL DEFAULT false
);
CREATE INDEX idx_orders_read_user       ON orders_read (user_id) WHERE is_deleted = false;
CREATE INDEX idx_orders_read_not_deleted ON orders_read (id)     WHERE is_deleted = false;
