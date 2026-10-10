DROP INDEX IF EXISTS idx_chat_messages_files;
ALTER TABLE chat_messages DROP COLUMN IF EXISTS file_expired;
