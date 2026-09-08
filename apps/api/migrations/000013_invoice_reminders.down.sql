DROP INDEX IF EXISTS idx_invoices_overdue_reminder;
ALTER TABLE invoices DROP COLUMN IF EXISTS reminder_sent_at;