CREATE INDEX ads_user_created_id_idx ON ads ((user_id::text), created_at DESC, id DESC);
CREATE INDEX ads_price_cents_idx ON ads (price_cents);
CREATE INDEX ads_created_id_idx ON ads (created_at DESC, id DESC);
