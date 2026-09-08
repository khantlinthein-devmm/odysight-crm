-- 000014: snapshot the customer email on bookings and invoices so invoice
-- emails can be delivered without a live FK to the customers table.

ALTER TABLE bookings
    ADD COLUMN customer_email TEXT NOT NULL DEFAULT '';

ALTER TABLE invoices
    ADD COLUMN customer_email TEXT NOT NULL DEFAULT '';