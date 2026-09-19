-- 000025: first-class Site entity.
--
-- A customer used to have exactly one address column, which cannot describe a
-- restaurant group with several branches or a condo with a lobby, gym and car
-- park. Sites are the physical places work happens; the customer stays the
-- commercial relationship.
--
-- Nothing becomes mandatory here: customers.address/area/property_type stay in
-- place and readable, and bookings.site_id is nullable so one-off bookings
-- (which need no customer at all) keep working exactly as before.

CREATE TABLE IF NOT EXISTS sites (
    id            BIGSERIAL PRIMARY KEY,
    customer_id   BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    address       TEXT NOT NULL DEFAULT '',
    area          TEXT NOT NULL DEFAULT '',
    property_type TEXT NOT NULL DEFAULT 'other'
        CHECK (property_type IN ('house', 'condo', 'apartment', 'office',
                                 'restaurant', 'factory', 'mall', 'retail', 'other')),
    -- Contact fields are site-level OVERRIDES only. Billing and the primary
    -- contact stay on the customer; blank means "use the customer's".
    contact_name  TEXT NOT NULL DEFAULT '',
    contact_phone TEXT NOT NULL DEFAULT '',
    contact_email TEXT NOT NULL DEFAULT '',
    notes         TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive')),
    lat           DOUBLE PRECISION,
    lng           DOUBLE PRECISION,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sites_customer ON sites (customer_id);
CREATE INDEX IF NOT EXISTS idx_sites_status ON sites (status);
CREATE INDEX IF NOT EXISTS idx_sites_area ON sites (area);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'sites_set_updated_at') THEN
    CREATE TRIGGER sites_set_updated_at BEFORE UPDATE ON sites FOR EACH ROW EXECUTE FUNCTION set_updated_at();
  END IF;
END $$;

-- Every existing customer gets one site carrying the address they already had,
-- so nothing is lost and multi-site customers can simply add more. Contact
-- override columns are deliberately left blank: the customer's own details
-- still apply.
INSERT INTO sites (customer_id, name, address, area, property_type)
SELECT c.id, 'Main site', c.address, c.area, c.property_type
  FROM customers c
 WHERE NOT EXISTS (SELECT 1 FROM sites s WHERE s.customer_id = c.id);

ALTER TABLE bookings ADD COLUMN IF NOT EXISTS site_id BIGINT REFERENCES sites(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_bookings_site_id ON bookings (site_id);

-- Attach historical bookings to their customer's site only where it is
-- unambiguous: the booking carries no address of its own, or exactly the one
-- the site was created from. Anything else stays NULL rather than risk
-- attributing work (and later revenue) to the wrong place.
UPDATE bookings b
   SET site_id = s.id
  FROM sites s
 WHERE b.site_id IS NULL
   AND b.customer_id IS NOT NULL
   AND s.customer_id = b.customer_id
   AND s.name = 'Main site'
   AND (b.address = '' OR b.address = s.address);
