-- 000034: customers created after 000025 but before the API began creating a
-- Default Site with each customer have no site at all. Give them one, the
-- same way 000025 backfilled older customers (idempotent).
INSERT INTO sites (customer_id, name, address, contact_name, phone, email, status, is_default)
SELECT id,
       'Default Site',
       COALESCE(NULLIF(address, ''), 'Address on file'),
       TRIM(first_name || ' ' || last_name),
       COALESCE(phone, ''),
       COALESCE(email, ''),
       'active',
       TRUE
  FROM customers c
 WHERE NOT EXISTS (SELECT 1 FROM sites s WHERE s.customer_id = c.id);
