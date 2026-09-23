-- 000025: commercial domain — sites, contracts, quotes, checklists.
--
-- Customer -> Sites -> Bookings, with optional Contract / Recurring Schedule /
-- Quotes / Checklists. Nothing here is mandatory for existing one-time flows:
-- bookings.site_id and bookings.contract_id are nullable, historical rows keep
-- working with free-text address.

-- Sites: physical locations where cleaning work happens.
CREATE TABLE IF NOT EXISTS sites (
    id            BIGSERIAL PRIMARY KEY,
    customer_id   BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    address       TEXT NOT NULL,
    contact_name  TEXT NOT NULL DEFAULT '',
    phone         TEXT NOT NULL DEFAULT '',
    email         TEXT NOT NULL DEFAULT '',
    notes         TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive')),
    latitude      DOUBLE PRECISION,
    longitude     DOUBLE PRECISION,
    is_default    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sites_customer_id ON sites (customer_id);
CREATE INDEX IF NOT EXISTS idx_sites_status ON sites (status);
-- At most one default site per customer (backfill target for legacy address).
CREATE UNIQUE INDEX IF NOT EXISTS uq_sites_customer_default
    ON sites (customer_id) WHERE is_default;

-- Backfill: every existing customer gets a Default Site seeded from its
-- current address/area so legacy customers/bookings keep working.
INSERT INTO sites (customer_id, name, address, contact_name, phone, email, status, is_default)
SELECT id,
       'Default Site',
       COALESCE(NULLIF(address, ''), 'Address on file'),
       TRIM(first_name || ' ' || last_name),
       COALESCE(phone, ''),
       COALESCE(email, ''),
       'active',
       TRUE
  FROM customers c
 WHERE NOT EXISTS (SELECT 1 FROM sites s WHERE s.customer_id = c.id);

-- Contracts: optional commercial agreements, possibly covering many sites.
CREATE TABLE IF NOT EXISTS contracts (
    id                BIGSERIAL PRIMARY KEY,
    contract_number   TEXT NOT NULL UNIQUE,
    customer_id       BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    title             TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'active', 'expiring', 'expired', 'cancelled', 'renewed')),
    start_date        DATE NOT NULL,
    end_date          DATE NOT NULL,
    renewal_date      DATE,
    contract_value    NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (contract_value >= 0),
    billing_frequency TEXT NOT NULL DEFAULT 'monthly'
        CHECK (billing_frequency IN ('monthly', 'quarterly', 'annual', 'custom')),
    sla_terms         TEXT NOT NULL DEFAULT '',
    notes             TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date)
);

CREATE SEQUENCE IF NOT EXISTS contract_seq;
CREATE INDEX IF NOT EXISTS idx_contracts_customer_id ON contracts (customer_id);
CREATE INDEX IF NOT EXISTS idx_contracts_status ON contracts (status);
CREATE INDEX IF NOT EXISTS idx_contracts_end_date ON contracts (end_date);

-- A contract may cover many sites; a site may be under many contracts over time.
CREATE TABLE IF NOT EXISTS contract_sites (
    contract_id BIGINT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    site_id     BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (contract_id, site_id)
);
CREATE INDEX IF NOT EXISTS idx_contract_sites_site_id ON contract_sites (site_id);

-- Bookings gain optional links. Nullable so one-time bookings keep working.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS site_id BIGINT REFERENCES sites(id) ON DELETE SET NULL;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS contract_id BIGINT REFERENCES contracts(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_bookings_site_id ON bookings (site_id);
CREATE INDEX IF NOT EXISTS idx_bookings_contract_id ON bookings (contract_id);

-- Quotes: persistent sales records. Never require a contract.
CREATE TABLE IF NOT EXISTS quotes (
    id                    BIGSERIAL PRIMARY KEY,
    quote_number          TEXT NOT NULL UNIQUE,
    customer_id           BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    site_id               BIGINT REFERENCES sites(id) ON DELETE SET NULL,
    status                TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'sent', 'accepted', 'rejected', 'expired')),
    valid_until           DATE,
    subtotal              NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (subtotal >= 0),
    tax_rate              NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (tax_rate >= 0 AND tax_rate <= 100),
    total                 NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (total >= 0),
    currency              CHAR(3) NOT NULL DEFAULT 'THB',
    notes                 TEXT NOT NULL DEFAULT '',
    version               INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    accepted_at           TIMESTAMPTZ,
    rejected_at           TIMESTAMPTZ,
    converted_booking_id  BIGINT REFERENCES bookings(id) ON DELETE SET NULL,
    converted_contract_id BIGINT REFERENCES contracts(id) ON DELETE SET NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS quote_seq;
CREATE INDEX IF NOT EXISTS idx_quotes_customer_id ON quotes (customer_id);
CREATE INDEX IF NOT EXISTS idx_quotes_site_id ON quotes (site_id);
CREATE INDEX IF NOT EXISTS idx_quotes_status ON quotes (status);

CREATE TABLE IF NOT EXISTS quote_items (
    id          BIGSERIAL PRIMARY KEY,
    quote_id    BIGINT NOT NULL REFERENCES quotes(id) ON DELETE CASCADE,
    service_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    quantity    NUMERIC(12,2) NOT NULL DEFAULT 1 CHECK (quantity >= 0),
    unit_price  NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
    line_total  NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (line_total >= 0),
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_quote_items_quote_id ON quote_items (quote_id);

-- Checklist templates reusable by service type.
CREATE TABLE IF NOT EXISTS checklist_templates (
    id           BIGSERIAL PRIMARY KEY,
    name         TEXT NOT NULL,
    service_type TEXT NOT NULL DEFAULT '',
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS checklist_template_items (
    id          BIGSERIAL PRIMARY KEY,
    template_id BIGINT NOT NULL REFERENCES checklist_templates(id) ON DELETE CASCADE,
    label       TEXT NOT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_checklist_template_items_template_id
    ON checklist_template_items (template_id);

-- Per-booking checklist instance (optional; at most one per booking).
CREATE TABLE IF NOT EXISTS booking_checklists (
    id                  BIGSERIAL PRIMARY KEY,
    booking_id          BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    template_id         BIGINT REFERENCES checklist_templates(id) ON DELETE SET NULL,
    status              TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'in_progress', 'completed')),
    client_signature    TEXT,
    client_confirmed_at TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (booking_id)
);
CREATE INDEX IF NOT EXISTS idx_booking_checklists_booking_id ON booking_checklists (booking_id);

CREATE TABLE IF NOT EXISTS booking_checklist_items (
    id            BIGSERIAL PRIMARY KEY,
    checklist_id  BIGINT NOT NULL REFERENCES booking_checklists(id) ON DELETE CASCADE,
    label         TEXT NOT NULL,
    is_completed  BOOLEAN NOT NULL DEFAULT FALSE,
    completed_by  TEXT,
    completed_at  TIMESTAMPTZ,
    notes         TEXT NOT NULL DEFAULT '',
    before_photo_url TEXT,
    after_photo_url  TEXT,
    sort_order    INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_booking_checklist_items_checklist_id
    ON booking_checklist_items (checklist_id);

-- Contract-aware invoicing: optional contract link + idempotency key so a
-- retried scheduler run can never bill the same period twice.
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS contract_id BIGINT REFERENCES contracts(id) ON DELETE SET NULL;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS billing_period_start DATE;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS billing_period_end DATE;
CREATE INDEX IF NOT EXISTS idx_invoices_contract_id ON invoices (contract_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_invoices_idempotency_key
    ON invoices (idempotency_key) WHERE idempotency_key IS NOT NULL;

-- updated_at triggers for the new tables.
DO $$
DECLARE t TEXT;
BEGIN
  FOREACH t IN ARRAY ARRAY['sites','contracts','quotes','checklist_templates','booking_checklists','booking_checklist_items'] LOOP
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = t || '_set_updated_at') THEN
      EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION set_updated_at()', t || '_set_updated_at', t);
    END IF;
  END LOOP;
END $$;
