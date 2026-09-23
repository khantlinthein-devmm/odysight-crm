-- 000027 down: relax the NOT NULL guard (data is kept as-is).

ALTER TABLE bookings ALTER COLUMN assigned_cleaner DROP NOT NULL;
ALTER TABLE bookings ALTER COLUMN assigned_cleaner DROP DEFAULT;
