-- Workspace settings key-value store + relax service_type / method enums
-- so the service catalog and payment methods are admin-editable.

CREATE TABLE IF NOT EXISTS settings (
  key        TEXT PRIMARY KEY,
  value      JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_service_type_check;
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_method_check;
