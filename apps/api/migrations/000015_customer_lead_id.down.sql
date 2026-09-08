DROP INDEX IF EXISTS idx_customers_lead_id_status;
DROP INDEX IF EXISTS idx_customers_lead_id;
ALTER TABLE customers DROP COLUMN lead_id;
