-- 000033: cleaning supplies — a catalog with stock, and a movement ledger
-- (purchases, usage at a site/job, adjustments) so supply cost can be
-- reported per site next to its revenue.

CREATE TABLE IF NOT EXISTS supplies (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    unit          TEXT NOT NULL DEFAULT 'piece',
    unit_cost     NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (unit_cost >= 0),
    stock_qty     NUMERIC(12,2) NOT NULL DEFAULT 0,
    reorder_level NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (reorder_level >= 0),
    active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_supplies_name ON supplies (lower(name));

CREATE TABLE IF NOT EXISTS supply_movements (
    id          BIGSERIAL PRIMARY KEY,
    supply_id   BIGINT NOT NULL REFERENCES supplies(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL CHECK (kind IN ('purchase', 'usage', 'adjustment')),
    -- Stock change: + for purchases, - for usage, either for adjustments.
    quantity    NUMERIC(12,2) NOT NULL CHECK (quantity <> 0),
    unit_cost   NUMERIC(12,2) NOT NULL DEFAULT 0,
    site_id     BIGINT REFERENCES sites(id) ON DELETE SET NULL,
    booking_id  BIGINT REFERENCES bookings(id) ON DELETE SET NULL,
    note        TEXT NOT NULL DEFAULT '',
    created_by  BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_supply_movements_site_time ON supply_movements (site_id, created_at);
CREATE INDEX IF NOT EXISTS idx_supply_movements_supply ON supply_movements (supply_id, created_at DESC);
