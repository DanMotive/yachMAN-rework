-- Cities
CREATE TABLE IF NOT EXISTS cities (
    id                 BIGSERIAL PRIMARY KEY,
    chat_id            BIGINT UNIQUE NOT NULL,
    name               TEXT NOT NULL,
    description        TEXT DEFAULT '',
    level              TEXT NOT NULL DEFAULT 'community',
    npc_population     INTEGER NOT NULL DEFAULT 0,
    development_points INTEGER NOT NULL DEFAULT 0,
    treasury           INTEGER NOT NULL DEFAULT 0,
    tax_rate_business  NUMERIC(5,2) NOT NULL DEFAULT 5.00,
    tax_rate_corporate NUMERIC(5,2) NOT NULL DEFAULT 8.00,
    tax_rate_income    NUMERIC(5,2) NOT NULL DEFAULT 2.00,
    last_tax_change_at TIMESTAMPTZ,
    access_mode        TEXT NOT NULL DEFAULT 'open',
    mayor_user_id      BIGINT,
    public_listing     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- City members
CREATE TABLE IF NOT EXISTS city_members (
    city_id   BIGINT NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
    user_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'resident',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (city_id, user_id)
);

-- City admins
CREATE TABLE IF NOT EXISTS city_admins (
    city_id      BIGINT NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    position     TEXT NOT NULL,
    appointed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (city_id, user_id)
);

-- City projects
CREATE TABLE IF NOT EXISTS city_projects (
    id          BIGSERIAL PRIMARY KEY,
    city_id     BIGINT NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT DEFAULT '',
    cost        INTEGER NOT NULL,
    progress    INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- City tax change audit log
CREATE TABLE IF NOT EXISTS city_taxes (
    id          BIGSERIAL PRIMARY KEY,
    city_id     BIGINT NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
    tax_type    TEXT NOT NULL,
    old_rate    NUMERIC(5,2),
    new_rate    NUMERIC(5,2),
    changed_by  BIGINT REFERENCES users(id),
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-city resource stock and demand tracking
CREATE TABLE IF NOT EXISTS city_resources (
    city_id     BIGINT NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
    resource_id TEXT NOT NULL REFERENCES resources(id),
    stock       INTEGER NOT NULL DEFAULT 0,
    demand      INTEGER NOT NULL DEFAULT 0,
    last_price  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (city_id, resource_id)
);

-- Foreign key: users.city_id -> cities.id
ALTER TABLE users ADD CONSTRAINT fk_users_city
    FOREIGN KEY (city_id) REFERENCES cities(id);

-- Foreign key: cities.mayor_user_id -> users.id
ALTER TABLE cities ADD CONSTRAINT fk_cities_mayor
    FOREIGN KEY (mayor_user_id) REFERENCES users(id);
