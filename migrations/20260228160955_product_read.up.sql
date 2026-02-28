CREATE TABLE product_read (
    id          BIGINT PRIMARY KEY,
    name  TEXT NOT NULL DEFAULT '',
    price  DOUBLE PRECISION NOT NULL DEFAULT 0,
    sku  TEXT NOT NULL DEFAULT '',
    active  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL,
    created_by  TEXT        NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    updated_by  TEXT        NOT NULL,
    is_deleted  BOOLEAN     NOT NULL DEFAULT false
);
CREATE INDEX idx_product_read_not_deleted ON product_read (id) WHERE is_deleted = false;
