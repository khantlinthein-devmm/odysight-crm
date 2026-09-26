ALTER TABLE invoices DROP COLUMN IF EXISTS withholding_amount;
ALTER TABLE invoices DROP COLUMN IF EXISTS withholding_rate;
ALTER TABLE invoices DROP COLUMN IF EXISTS customer_tax_branch;
ALTER TABLE invoices DROP COLUMN IF EXISTS customer_tax_id;

ALTER TABLE customers DROP COLUMN IF EXISTS withholding_rate;
ALTER TABLE customers DROP COLUMN IF EXISTS tax_branch;
ALTER TABLE customers DROP COLUMN IF EXISTS tax_id;
