-- 000024: stop the recurring generator producing duplicate occurrences.
--
-- The generator now inserts the successor and retires the source in one
-- transaction. This index is the backstop: a retry, or a second scheduler
-- instance, can never write the same occurrence twice.
--
-- Duplicates the previous non-transactional generator may already have created
-- would block the index, so they are detached from their series first. They
-- are NOT deleted or cancelled: they may be real bookings someone has worked
-- and invoiced, so they stay visible and billable and simply stop taking part
-- in the series. The earliest row of each slot keeps its series_id.

UPDATE bookings b
   SET series_id = NULL
 WHERE b.series_id IS NOT NULL
   AND EXISTS (
       SELECT 1
         FROM bookings keep
        WHERE keep.series_id = b.series_id
          AND keep.scheduled_for = b.scheduled_for
          AND keep.id < b.id
   );

CREATE UNIQUE INDEX IF NOT EXISTS uq_bookings_series_slot
    ON bookings (series_id, scheduled_for) WHERE series_id IS NOT NULL;
