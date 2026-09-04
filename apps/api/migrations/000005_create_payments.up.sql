CREATE TABLE IF NOT EXISTS payments (
    id             BIGSERIAL PRIMARY KEY,
    invoice_number TEXT NOT NULL UNIQUE,
    payer_name     TEXT NOT NULL,
    amount         NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    currency       CHAR(3) NOT NULL,
    method         TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'paid', 'failed', 'refunded')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payments_status ON payments (status);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments (created_at DESC);
