-- 000023: LINE ID for cleaners, and make last name / email optional across
-- cleaners, customers and leads. Walk-in customers and field staff often have
-- neither, so both are stored as '' rather than being required.

ALTER TABLE cleaners ADD COLUMN IF NOT EXISTS line_id TEXT NOT NULL DEFAULT '';

ALTER TABLE cleaners  ALTER COLUMN last_name SET DEFAULT '';
ALTER TABLE cleaners  ALTER COLUMN email     SET DEFAULT '';
ALTER TABLE customers ALTER COLUMN last_name SET DEFAULT '';
ALTER TABLE customers ALTER COLUMN email     SET DEFAULT '';
ALTER TABLE leads     ALTER COLUMN last_name SET DEFAULT '';
ALTER TABLE leads     ALTER COLUMN email     SET DEFAULT '';

-- Many records may now share a blank email, so uniqueness applies only to
-- real addresses.
DROP INDEX IF EXISTS idx_cleaners_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_cleaners_email
    ON cleaners (email) WHERE email <> '';

DROP INDEX IF EXISTS idx_customers_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_email
    ON customers (email) WHERE email <> '';

DROP INDEX IF EXISTS idx_leads_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_leads_email
    ON leads (email) WHERE email <> '';
