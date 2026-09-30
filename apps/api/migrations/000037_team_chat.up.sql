-- 000037: team chat (direct messages).
-- Admins put team members into chat groups; two people may message each
-- other when they share a group, and office roles can message anyone.

CREATE TABLE IF NOT EXISTS chat_groups (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS chat_group_members (
    group_id BIGINT NOT NULL REFERENCES chat_groups(id) ON DELETE CASCADE,
    user_id  BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_chat_group_members_user ON chat_group_members (user_id);

-- One conversation per pair of users; user_a < user_b keeps it unique.
CREATE TABLE IF NOT EXISTS chat_conversations (
    id              BIGSERIAL PRIMARY KEY,
    user_a          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read_a     BIGINT NOT NULL DEFAULT 0,
    last_read_b     BIGINT NOT NULL DEFAULT 0,
    last_message_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (user_a < user_b),
    UNIQUE (user_a, user_b)
);
CREATE INDEX IF NOT EXISTS idx_chat_conversations_b ON chat_conversations (user_b);

CREATE TABLE IF NOT EXISTS chat_messages (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    sender_id       BIGINT REFERENCES users(id) ON DELETE SET NULL,
    kind            TEXT NOT NULL CHECK (kind IN ('text', 'image', 'voice')),
    body            TEXT NOT NULL DEFAULT '',
    file_name       TEXT NOT NULL DEFAULT '',
    mime_type       TEXT NOT NULL DEFAULT '',
    duration_ms     INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation ON chat_messages (conversation_id, id);

-- Browser push subscriptions (one per device / browser).
CREATE TABLE IF NOT EXISTS push_subscriptions (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint   TEXT NOT NULL UNIQUE,
    p256dh     TEXT NOT NULL,
    auth       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user ON push_subscriptions (user_id);

-- The server's VAPID key pair, generated on first start.
CREATE TABLE IF NOT EXISTS push_vapid_keys (
    id          SMALLINT PRIMARY KEY CHECK (id = 1),
    private_key TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
