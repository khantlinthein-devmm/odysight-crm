-- Reverts 000024. The series_id values cleared by the up migration cannot be
-- restored: the duplicate rows they identified are indistinguishable from
-- ordinary one-off bookings once detached.

DROP INDEX IF EXISTS uq_bookings_series_slot;
