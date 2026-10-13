-- 000047: one cleaners fee for the whole job team (e.g. 2,000 for the lead
-- and everyone who came), instead of a fee per cleaner. NULL = fees are
-- entered per cleaner in cleaner_fees.
ALTER TABLE booking_costs ADD COLUMN IF NOT EXISTS team_fee NUMERIC(12,2) CHECK (team_fee >= 0);
