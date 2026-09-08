-- 000012: invoices — bill completed bookings from the service catalog.

CREATE TABLE IF NOT EXISTS invoices (
    id             BIGSERIAL PRIMARY KEY,
    invoice_number TEXT NOT NULL UNIQUE,
    booking_id     BIGINT NOT NULL REFERENCES bookings(id) ON DELETE RESTRICT,
    booking_number TEXT NOT NULL,
    customer_name  TEXT NOT NULL,
    address        TEXT NOT NULL DEFAULT '',
    service_type   TEXT NOT NULL,
    service_name   TEXT NOT NULL,
    subtotal       NUMERIC(12,2) NOT NULL CHECK (subtotal >= 0),
    tax_rate       NUMERIC(5,2) NOT NULL DEFAULT 0
        CHECK (tax_rate >= 0 AND tax_rate <= 100),
    tax_amount     NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    total          NUMERIC(12,2) NOT NULL CHECK (total >= 0),
    currency       CHAR(3) NOT NULL DEFAULT 'THB',
    status         TEXT NOT NULL DEFAULT 'issued'
        CHECK (status IN ('draft', 'issued', 'paid', 'void')),
    issued_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS invoice_seq;

CREATE INDEX IF NOT EXISTS idx_invoices_status        ON invoices (status);
CREATE INDEX IF NOT EXISTS idx_invoices_booking_id    ON invoices (booking_id);
CREATE INDEX IF NOT EXISTS idx_invoices_customer_name ON invoices (customer_name);
CREATE INDEX IF NOT EXISTS idx_invoices_created_at    ON invoices (created_at DESC);

-- At most one active (non-void) invoice per booking.
CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_booking_active
    ON invoices (booking_id) WHERE status <> 'void';

DROP TRIGGER IF EXISTS invoices_set_updated_at ON invoices;
CREATE TRIGGER invoices_set_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();