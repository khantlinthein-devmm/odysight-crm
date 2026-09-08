-- 000013: let invoices survive booking deletion (booking_id becomes NULL,
-- invoice keeps booking_number as text). Dereferenced invoices stay visible
-- for history and can still be voided.

ALTER TABLE invoices DROP CONSTRAINT invoices_booking_id_fkey;
ALTER TABLE invoices ALTER COLUMN booking_id DROP NOT NULL;
ALTER TABLE invoices
    ADD CONSTRAINT invoices_booking_id_fkey
    FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE SET NULL;