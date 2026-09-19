-- 000022: extend attendance to office/team staff, not just cleaners.
-- A row belongs to exactly one person: a cleaner (field staff) OR a user
-- (office staff), never both and never neither.

ALTER TABLE attendance ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE attendance ALTER COLUMN cleaner_id DROP NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'attendance_person_check') THEN
    ALTER TABLE attendance ADD CONSTRAINT attendance_person_check
      CHECK ((cleaner_id IS NOT NULL) <> (user_id IS NOT NULL));
  END IF;
END $$;

-- A table-wide UNIQUE no longer fits a nullable cleaner_id, so each person
-- type gets a partial unique index: one row per person per work date.
ALTER TABLE attendance DROP CONSTRAINT IF EXISTS uq_attendance_cleaner_day;

CREATE UNIQUE INDEX IF NOT EXISTS uq_attendance_cleaner_day
    ON attendance (cleaner_id, work_date) WHERE cleaner_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_attendance_user_day
    ON attendance (user_id, work_date) WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_attendance_user ON attendance (user_id);
