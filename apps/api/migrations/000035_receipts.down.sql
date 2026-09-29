DROP TABLE IF EXISTS receipts;
DROP SEQUENCE IF EXISTS receipt_seq;
ALTER TABLE bookings DROP COLUMN IF EXISTS price;
DROP INDEX IF EXISTS idx_payments_invoice_id;
ALTER TABLE payments DROP COLUMN IF EXISTS reference;
ALTER TABLE payments DROP COLUMN IF EXISTS invoice_id;
DROP INDEX IF EXISTS idx_payments_invoice_number;
UPDATE invoices SET status = 'issued' WHERE status = 'partially_paid';
ALTER TABLE invoices DROP COLUMN IF EXISTS amount_paid;
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_status_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_status_check
    CHECK (status IN ('draft', 'issued', 'paid', 'void'));
