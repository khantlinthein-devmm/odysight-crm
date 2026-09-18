-- 000019: daily staff attendance (check-in / check-out per cleaner per day).

CREATE TABLE IF NOT EXISTS attendance (
    id            BIGSERIAL PRIMARY KEY,
    cleaner_id    BIGINT NOT NULL REFERENCES cleaners(id) ON DELETE CASCADE,
    work_date     DATE NOT NULL,
    check_in_at   TIMESTAMPTZ,
    check_out_at  TIMESTAMPTZ,
    note          TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_attendance_cleaner_day UNIQUE (cleaner_id, work_date)
);

CREATE INDEX IF NOT EXISTS idx_attendance_date ON attendance (work_date DESC);
CREATE INDEX IF NOT EXISTS idx_attendance_cleaner ON attendance (cleaner_id);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'attendance_set_updated_at') THEN
    CREATE TRIGGER attendance_set_updated_at BEFORE UPDATE ON attendance FOR EACH ROW EXECUTE FUNCTION set_updated_at();
  END IF;
END $$;
