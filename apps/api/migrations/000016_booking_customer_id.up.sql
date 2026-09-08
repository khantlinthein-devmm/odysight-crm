-- 000016: link bookings to the customers table. The denormalized
-- customer_name stays as the display snapshot (invoices copy it), while
-- customer_id records which customer the booking belongs to (NULL = manual).

ALTER TABLE bookings
    ADD COLUMN customer_id BIGINT REFERENCES customers(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_bookings_customer_id ON bookings (customer_id);