DROP TRIGGER IF EXISTS invoices_set_updated_at ON invoices;
DROP INDEX IF EXISTS idx_invoices_booking_active;
DROP INDEX IF EXISTS idx_invoices_created_at;
DROP INDEX IF EXISTS idx_invoices_customer_name;
DROP INDEX IF EXISTS idx_invoices_booking_id;
DROP INDEX IF EXISTS idx_invoices_status;
DROP SEQUENCE IF EXISTS invoice_seq;
DROP TABLE IF EXISTS invoices;