CREATE TABLE IF NOT EXISTS leads (
    id         BIGSERIAL PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name  TEXT NOT NULL,
    email      TEXT NOT NULL,
    phone      TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'contacted', 'quote_sent', 'booked', 'won', 'lost')),
    source     TEXT NOT NULL
        CHECK (source IN ('website', 'referral', 'line', 'facebook', 'walk_in', 'campaign')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_leads_status ON leads (status);
CREATE INDEX IF NOT EXISTS idx_leads_created_at ON leads (created_at DESC);
