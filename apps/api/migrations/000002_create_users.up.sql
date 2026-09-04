CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL
        CHECK (role IN ('SUPER_ADMIN', 'ADMIN', 'MANAGER', 'SALES', 'STAFF', 'ACCOUNTANT')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
