-- 000025 down: remove commercial domain. Historical bookings/invoices keep
-- working because their new columns are dropped, not their rows.

DROP TRIGGER IF EXISTS booking_checklist_items_set_updated_at ON booking_checklist_items;
DROP TRIGGER IF EXISTS booking_checklists_set_updated_at ON booking_checklists;
DROP TRIGGER IF EXISTS checklist_templates_set_updated_at ON checklist_templates;
DROP TRIGGER IF EXISTS quotes_set_updated_at ON quotes;
DROP TRIGGER IF EXISTS contracts_set_updated_at ON contracts;
DROP TRIGGER IF EXISTS sites_set_updated_at ON sites;

DROP INDEX IF EXISTS uq_invoices_idempotency_key;
DROP INDEX IF EXISTS idx_invoices_contract_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS billing_period_end;
ALTER TABLE invoices DROP COLUMN IF EXISTS billing_period_start;
ALTER TABLE invoices DROP COLUMN IF EXISTS idempotency_key;
ALTER TABLE invoices DROP COLUMN IF EXISTS contract_id;

DROP TABLE IF EXISTS booking_checklist_items;
DROP TABLE IF EXISTS booking_checklists;
DROP TABLE IF EXISTS checklist_template_items;
DROP TABLE IF EXISTS checklist_templates;
DROP TABLE IF EXISTS quote_items;
DROP TABLE IF EXISTS quotes;

DROP INDEX IF EXISTS idx_bookings_contract_id;
DROP INDEX IF EXISTS idx_bookings_site_id;
ALTER TABLE bookings DROP COLUMN IF EXISTS contract_id;
ALTER TABLE bookings DROP COLUMN IF EXISTS site_id;

DROP TABLE IF EXISTS contract_sites;
DROP TABLE IF EXISTS contracts;
DROP TABLE IF EXISTS sites;
