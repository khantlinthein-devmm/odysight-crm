DROP INDEX IF EXISTS idx_bookings_customer_id;
ALTER TABLE bookings DROP COLUMN customer_id;