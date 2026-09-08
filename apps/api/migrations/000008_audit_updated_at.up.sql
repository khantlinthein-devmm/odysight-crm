-- Add updated_at columns + triggers, audit log table.

ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE leads ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE customers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE cleaners ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE payments ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE service_records ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
DECLARE t TEXT;
BEGIN
  FOREACH t IN ARRAY ARRAY['users','leads','customers','cleaners','bookings','payments','service_records'] LOOP
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = t || '_set_updated_at') THEN
      EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION set_updated_at()', t || '_set_updated_at', t);
    END IF;
  END LOOP;
END $$;

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT,
  action TEXT NOT NULL,
  resource TEXT NOT NULL,
  resource_id BIGINT,
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs (user_id);
