-- 000014 down

ALTER TABLE invoices
    DROP COLUMN customer_email;

ALTER TABLE bookings
    DROP COLUMN customer_email;