-- 000028: Thai tax invoice support.
-- Customers carry their tax identity (13-digit TIN + branch) and a default
-- withholding-tax rate (companies deduct 3% on cleaning services). Invoices
-- snapshot those at issue time so a later customer edit never rewrites a
-- tax document that has already been issued.

ALTER TABLE customers ADD COLUMN IF NOT EXISTS tax_id TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS tax_branch TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS withholding_rate NUMERIC(5,2) NOT NULL DEFAULT 0
    CHECK (withholding_rate >= 0 AND withholding_rate <= 15);

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS customer_tax_id TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS customer_tax_branch TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS withholding_rate NUMERIC(5,2) NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS withholding_amount NUMERIC(14,2) NOT NULL DEFAULT 0;
