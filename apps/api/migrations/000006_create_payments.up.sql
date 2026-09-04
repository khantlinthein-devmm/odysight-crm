CREATE TABLE IF NOT EXISTS payments (
    id             BIGSERIAL PRIMARY KEY,
    invoice_number TEXT NOT NULL UNIQUE,
    customer_name  TEXT NOT NULL,
    booking_number TEXT NOT NULL DEFAULT '',
    amount         NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    currency       CHAR(3) NOT NULL DEFAULT 'THB',
    method         TEXT NOT NULL
        CHECK (method IN ('cash', 'bank_transfer', 'promptpay', 'credit_card', 'line_pay', 'online_wallet')),
    status         TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'paid', 'failed', 'refunded')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payments_status ON payments (status);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments (created_at DESC);
