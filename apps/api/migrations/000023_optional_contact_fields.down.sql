-- Reverts 000023. Rows left with a blank email would violate the restored
-- table-wide unique index, so they are given a placeholder address first.

UPDATE cleaners  SET email = 'no-email-' || id || '@invalid' WHERE email = '';
UPDATE customers SET email = 'no-email-' || id || '@invalid' WHERE email = '';
UPDATE leads     SET email = 'no-email-' || id || '@invalid' WHERE email = '';

DROP INDEX IF EXISTS idx_cleaners_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_cleaners_email ON cleaners (email);

DROP INDEX IF EXISTS idx_customers_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_email ON customers (email);

DROP INDEX IF EXISTS idx_leads_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_leads_email ON leads (email);

ALTER TABLE cleaners  ALTER COLUMN last_name DROP DEFAULT;
ALTER TABLE cleaners  ALTER COLUMN email     DROP DEFAULT;
ALTER TABLE customers ALTER COLUMN last_name DROP DEFAULT;
ALTER TABLE customers ALTER COLUMN email     DROP DEFAULT;
ALTER TABLE leads     ALTER COLUMN last_name DROP DEFAULT;
ALTER TABLE leads     ALTER COLUMN email     DROP DEFAULT;

ALTER TABLE cleaners DROP COLUMN IF EXISTS line_id;
