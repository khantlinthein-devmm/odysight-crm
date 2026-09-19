-- Reverts 000025. customers.address/area/property_type were never removed, so
-- dropping sites loses only the extra sites a customer gained after the split.

DROP INDEX IF EXISTS idx_bookings_site_id;
ALTER TABLE bookings DROP COLUMN IF EXISTS site_id;

DROP TABLE IF EXISTS sites;
