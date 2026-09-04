CREATE TABLE IF NOT EXISTS customers (
    id            BIGSERIAL PRIMARY KEY,
    first_name    TEXT NOT NULL,
    last_name     TEXT NOT NULL,
    email         TEXT NOT NULL,
    phone         TEXT NOT NULL,
    address       TEXT NOT NULL,
    property_type TEXT NOT NULL
        CHECK (property_type IN ('house', 'condo', 'office', 'apartment', 'other')),
    area          TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'blocked')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_customers_status ON customers (status);
CREATE INDEX IF NOT EXISTS idx_customers_created_at ON customers (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_customers_area ON customers (area);
