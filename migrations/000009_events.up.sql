-- Events (world, city, economic, social)
CREATE TABLE IF NOT EXISTS events (
    id          BIGSERIAL PRIMARY KEY,
    type        TEXT NOT NULL,
    name        TEXT NOT NULL,
    description TEXT DEFAULT '',
    city_id     BIGINT REFERENCES cities(id),
    start_at    TIMESTAMPTZ NOT NULL,
    end_at      TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_active ON events (start_at, end_at);

-- Event effects (modifiers)
CREATE TABLE IF NOT EXISTS event_effects (
    id          BIGSERIAL PRIMARY KEY,
    event_id    BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL,
    target_id   TEXT,
    modifier    NUMERIC(5,2) NOT NULL
);
