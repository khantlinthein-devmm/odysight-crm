-- Reverts 000022. Staff attendance rows are dropped: cleaner_id cannot go back
-- to NOT NULL while they exist.

DROP INDEX IF EXISTS idx_attendance_user;
DROP INDEX IF EXISTS uq_attendance_user_day;
DROP INDEX IF EXISTS uq_attendance_cleaner_day;

DELETE FROM attendance WHERE user_id IS NOT NULL;

ALTER TABLE attendance DROP CONSTRAINT IF EXISTS attendance_person_check;
ALTER TABLE attendance DROP COLUMN IF EXISTS user_id;
ALTER TABLE attendance ALTER COLUMN cleaner_id SET NOT NULL;

ALTER TABLE attendance ADD CONSTRAINT uq_attendance_cleaner_day UNIQUE (cleaner_id, work_date);
