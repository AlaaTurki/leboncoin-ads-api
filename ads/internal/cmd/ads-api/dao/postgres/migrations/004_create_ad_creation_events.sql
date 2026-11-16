CREATE TABLE ad_creation_events (
    id         BIGSERIAL PRIMARY KEY,
    user_id    UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ad_creation_events_user_created_idx ON ad_creation_events (user_id, created_at DESC);
