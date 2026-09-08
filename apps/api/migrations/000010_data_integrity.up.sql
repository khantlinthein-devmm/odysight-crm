-- 000010: data integrity — dedupe + unique indexes on business-key emails,
-- plus indexes on frequently-joined/filtered columns.

-- Deduplicate leads by email (keep the lowest id) before enforcing uniqueness.
DELETE FROM leads a USING leads b WHERE a.id > b.id AND a.email = b.email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_leads_email ON leads (email);

-- Deduplicate customers by email.
DELETE FROM customers a USING customers b WHERE a.id > b.id AND a.email = b.email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_email ON customers (email);

-- Deduplicate cleaners by email.
DELETE FROM cleaners a USING cleaners b WHERE a.id > b.id AND a.email = b.email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_cleaners_email ON cleaners (email);

-- Frequently filtered/joined columns currently lacking indexes.
CREATE INDEX IF NOT EXISTS idx_service_records_booking_number ON service_records (booking_number);
CREATE INDEX IF NOT EXISTS idx_service_records_cleaner_name ON service_records (cleaner_name);
CREATE INDEX IF NOT EXISTS idx_bookings_customer_name ON bookings (customer_name);
CREATE INDEX IF NOT EXISTS idx_bookings_assigned_cleaner ON bookings (assigned_cleaner);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON audit_logs (resource_id);