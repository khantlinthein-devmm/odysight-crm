-- 000029: LINE messages to customers.
-- A customer converted from a LINE lead inherits that lead's LINE user id,
-- so booking confirmations, reminders, completion notices and invoices can
-- be pushed to the chat they started in. The *_at columns make the reminder
-- and completion pushes fire at most once per booking.

ALTER TABLE customers ADD COLUMN IF NOT EXISTS line_user_id TEXT NOT NULL DEFAULT '';

UPDATE customers c
   SET line_user_id = l.line_user_id
  FROM leads l
 WHERE c.lead_id = l.id AND c.line_user_id = '' AND l.line_user_id <> '';

ALTER TABLE bookings ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMPTZ;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS completion_notified_at TIMESTAMPTZ;

-- Existing completed jobs must not get a late "job done" push.
UPDATE bookings SET completion_notified_at = now()
 WHERE status = 'completed' AND completion_notified_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_bookings_reminder_due
    ON bookings (scheduled_for)
 WHERE reminder_sent_at IS NULL AND status IN ('pending', 'confirmed');
