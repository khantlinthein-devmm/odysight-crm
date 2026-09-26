-- 000030: GPS on check-in/out. Cleaners' phones send their position; the
-- API checks it against the day's job sites (when those sites have
-- coordinates) and stores where each stamp was made and how far from the
-- nearest site, so the office can audit attendance.

ALTER TABLE attendance ADD COLUMN IF NOT EXISTS check_in_lat DOUBLE PRECISION;
ALTER TABLE attendance ADD COLUMN IF NOT EXISTS check_in_lng DOUBLE PRECISION;
ALTER TABLE attendance ADD COLUMN IF NOT EXISTS check_out_lat DOUBLE PRECISION;
ALTER TABLE attendance ADD COLUMN IF NOT EXISTS check_out_lng DOUBLE PRECISION;
ALTER TABLE attendance ADD COLUMN IF NOT EXISTS check_in_site_id BIGINT REFERENCES sites(id) ON DELETE SET NULL;
ALTER TABLE attendance ADD COLUMN IF NOT EXISTS check_in_distance_m INTEGER;
