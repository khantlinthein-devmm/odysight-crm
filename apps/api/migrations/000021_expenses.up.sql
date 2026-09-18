-- 000021: office expenses ledger (powers the Finance income/expenses page).

CREATE TABLE IF NOT EXISTS expenses (
    id         BIGSERIAL PRIMARY KEY,
    spent_on   DATE NOT NULL,
    category   TEXT NOT NULL DEFAULT 'Other',
    amount     NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    note       TEXT NOT NULL DEFAULT '',
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_expenses_spent_on ON expenses (spent_on DESC);
CREATE INDEX IF NOT EXISTS idx_expenses_category ON expenses (category);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'expenses_set_updated_at') THEN
    CREATE TRIGGER expenses_set_updated_at BEFORE UPDATE ON expenses FOR EACH ROW EXECUTE FUNCTION set_updated_at();
  END IF;
END $$;
