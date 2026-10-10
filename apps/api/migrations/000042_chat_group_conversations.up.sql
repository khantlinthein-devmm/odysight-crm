-- 000042: group chat.
-- Every chat group also has one group conversation that all its members
-- share. Direct conversations keep user_a/user_b; a group conversation has
-- group_id instead, and per-member read markers live in chat_reads.

ALTER TABLE chat_conversations ALTER COLUMN user_a DROP NOT NULL;
ALTER TABLE chat_conversations ALTER COLUMN user_b DROP NOT NULL;
ALTER TABLE chat_conversations ADD COLUMN IF NOT EXISTS group_id BIGINT UNIQUE REFERENCES chat_groups(id) ON DELETE CASCADE;
ALTER TABLE chat_conversations DROP CONSTRAINT IF EXISTS chat_conversations_check;
ALTER TABLE chat_conversations DROP CONSTRAINT IF EXISTS chat_conversations_kind;
ALTER TABLE chat_conversations ADD CONSTRAINT chat_conversations_kind CHECK (
    (group_id IS NULL AND user_a IS NOT NULL AND user_b IS NOT NULL AND user_a < user_b)
 OR (group_id IS NOT NULL AND user_a IS NULL AND user_b IS NULL));

CREATE TABLE IF NOT EXISTS chat_reads (
    conversation_id BIGINT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read       BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (conversation_id, user_id)
);

INSERT INTO chat_conversations (group_id)
SELECT id FROM chat_groups ON CONFLICT (group_id) DO NOTHING;
