-- 000013 down: restore RESTRICT semantics; orphaned invoices cannot be kept.

ALTER TABLE invoices DROP CONSTRAINT invoices_booking_id_fkey;
DELETE FROM invoices WHERE booking_id IS NULL;
ALTER TABLE invoices ALTER COLUMN booking_id SET NOT NULL;
ALTER TABLE invoices
    ADD CONSTRAINT invoices_booking_id_fkey
    FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE RESTRICT;