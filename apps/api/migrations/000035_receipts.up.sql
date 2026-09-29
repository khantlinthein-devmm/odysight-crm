-- 000035: receipts and partial payments.
-- An invoice (ใบแจ้งหนี้) now tracks how much has been received; every paid
-- payment against it issues its own receipt (ใบเสร็จรับเงิน, or
-- ใบเสร็จรับเงิน/ใบกำกับภาษี when the company is VAT registered). Payments
-- link to their invoice by id, and one invoice can take several payments.

ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_status_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_status_check
    CHECK (status IN ('draft', 'issued', 'partially_paid', 'paid', 'void'));
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS amount_paid NUMERIC(14,2) NOT NULL DEFAULT 0
    CHECK (amount_paid >= 0);
-- Invoices settled before this migration were paid in full.
UPDATE invoices SET amount_paid = total - withholding_amount
 WHERE status = 'paid' AND amount_paid = 0;

-- A payment reference used to be unique per invoice number, which made a
-- second (partial) payment against the same invoice impossible.
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_invoice_number_key;
CREATE INDEX IF NOT EXISTS idx_payments_invoice_number ON payments (invoice_number);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS invoice_id BIGINT REFERENCES invoices(id) ON DELETE SET NULL;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS reference TEXT NOT NULL DEFAULT '';
UPDATE payments p SET invoice_id = i.id
  FROM invoices i
 WHERE p.invoice_id IS NULL AND i.invoice_number = p.invoice_number;
CREATE INDEX IF NOT EXISTS idx_payments_invoice_id ON payments (invoice_id);

-- The agreed job price (before VAT). NULL falls back to the catalog price.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS price NUMERIC(12,2)
    CHECK (price IS NULL OR price >= 0);

CREATE SEQUENCE IF NOT EXISTS receipt_seq;

-- One receipt per paid payment. The amounts split the cash received into
-- the service value, VAT and withholding tax it settles, so a partial payment
-- still yields a correct tax document. Customer details are read from the
-- invoice, which is an immutable snapshot.
CREATE TABLE IF NOT EXISTS receipts (
    id               BIGSERIAL PRIMARY KEY,
    receipt_number   TEXT NOT NULL UNIQUE,
    invoice_id       BIGINT NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT,
    payment_id       BIGINT NOT NULL UNIQUE REFERENCES payments(id) ON DELETE RESTRICT,
    amount           NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    subtotal_part    NUMERIC(14,2) NOT NULL DEFAULT 0,
    vat_part         NUMERIC(14,2) NOT NULL DEFAULT 0,
    wht_part         NUMERIC(14,2) NOT NULL DEFAULT 0,
    tax_rate         NUMERIC(5,2) NOT NULL DEFAULT 0,
    withholding_rate NUMERIC(5,2) NOT NULL DEFAULT 0,
    currency         CHAR(3) NOT NULL DEFAULT 'THB',
    method           TEXT NOT NULL,
    reference        TEXT NOT NULL DEFAULT '',
    vat_registered   BOOLEAN NOT NULL DEFAULT false,
    status           TEXT NOT NULL DEFAULT 'valid' CHECK (status IN ('valid', 'cancelled')),
    paid_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    cancelled_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_receipts_invoice_id ON receipts (invoice_id);
CREATE INDEX IF NOT EXISTS idx_receipts_created_at ON receipts (created_at DESC);
