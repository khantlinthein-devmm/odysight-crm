-- 000013: track when overdue invoice reminders are emailed to customers.

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMPTZ;

-- The reminder scan targets one narrow slice: issued, unpaid invoices that have
-- never been reminded. The partial index covers exactly that predicate.
CREATE INDEX IF NOT EXISTS idx_invoices_overdue_reminder
    ON invoices (issued_at)
    WHERE status = 'issued' AND paid_at IS NULL AND reminder_sent_at IS NULL;