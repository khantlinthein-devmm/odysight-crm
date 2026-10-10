-- 000040: one-time catch-up for customers renamed before edits started
-- propagating: copy each customer's current name and email onto their
-- bookings, those bookings' invoices (receipts read the invoice) and
-- payments. From now on the customer update keeps them in step.
UPDATE bookings b
   SET customer_name = TRIM(c.first_name || ' ' || c.last_name), customer_email = c.email
  FROM customers c
 WHERE b.customer_id = c.id
   AND (b.customer_name IS DISTINCT FROM TRIM(c.first_name || ' ' || c.last_name)
        OR b.customer_email IS DISTINCT FROM c.email);

UPDATE invoices i
   SET customer_name = b.customer_name, customer_email = b.customer_email
  FROM bookings b
 WHERE i.booking_id = b.id AND b.customer_id IS NOT NULL
   AND (i.customer_name IS DISTINCT FROM b.customer_name OR i.customer_email IS DISTINCT FROM b.customer_email);

UPDATE payments p
   SET customer_name = b.customer_name
  FROM bookings b
 WHERE p.booking_number = b.booking_number AND b.customer_id IS NOT NULL
   AND p.customer_name IS DISTINCT FROM b.customer_name;
