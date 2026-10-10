-- 000043: quotations whose line prices already include VAT (ราคารวม VAT).
-- subtotal is then the value before VAT and total the sum of the lines.
ALTER TABLE quotes ADD COLUMN IF NOT EXISTS prices_include_vat BOOLEAN NOT NULL DEFAULT false;
