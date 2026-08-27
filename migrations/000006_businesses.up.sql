-- Business type definitions (seed-loaded, 65 types)
CREATE TABLE IF NOT EXISTS business_types (
    type_id        TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    input_a_resource TEXT NOT NULL,
    input_a_amount INTEGER NOT NULL,
    input_b_resource TEXT NOT NULL,
    input_b_amount INTEGER NOT NULL,
    output_resource TEXT NOT NULL,
    output_amount  INTEGER NOT NULL,
    npc_staff      INTEGER NOT NULL
);

-- Business instances in cities
CREATE TABLE IF NOT EXISTS businesses (
    id             BIGSERIAL PRIMARY KEY,
    city_id        BIGINT NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
    type_id        TEXT NOT NULL REFERENCES business_types(type_id),
    name           TEXT NOT NULL,
    owner_user_id  BIGINT REFERENCES users(id),
    corporation_id BIGINT REFERENCES corporations(id),
    power_pct      INTEGER NOT NULL DEFAULT 100,
    last_tick_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_businesses_city ON businesses (city_id);
CREATE INDEX idx_businesses_owner ON businesses (owner_user_id);
CREATE INDEX idx_businesses_corporation ON businesses (corporation_id);

-- Business staff (real players in corporate businesses)
CREATE TABLE IF NOT EXISTS business_staff (
    id          BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    position    TEXT NOT NULL,
    salary      INTEGER NOT NULL DEFAULT 0,
    hired_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_business_staff_unique ON business_staff (business_id, user_id);
