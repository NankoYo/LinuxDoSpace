-- 017_domain_purchase_reservations.sql adds one exact-purchase reservation key
-- so concurrent checkout creation cannot oversell the same namespace, and it
-- also lifts the default per-root-domain sale price to 10 LDC.

ALTER TABLE payment_orders
    ADD COLUMN purchase_reservation_key TEXT NOT NULL DEFAULT '';

ALTER TABLE payment_orders
    ADD COLUMN purchase_reservation_expires_at TEXT NULL;

UPDATE managed_domains
SET sale_base_price_cents = 1000
WHERE sale_base_price_cents = 0;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_live_exact_purchase_reservation
ON payment_orders(purchase_reservation_key)
WHERE purchase_reservation_key <> ''
  AND status IN ('created', 'pending', 'paid');
