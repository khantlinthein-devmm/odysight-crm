-- 000015: link customers back to the lead they were converted from.
-- A customer may be created manually (lead_id NULL) or converted from a lead.

ALTER TABLE customers
    ADD COLUMN lead_id BIGINT REFERENCES leads(id) ON DELETE SET NULL;

-- A lead should convert into at most one customer.
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_lead_id
    ON customers (lead_id)
    WHERE lead_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_customers_lead_id_status
    ON customers (lead_id, status);
