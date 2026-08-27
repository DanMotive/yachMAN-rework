-- Users table
CREATE TABLE IF NOT EXISTS users (
    id              BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT UNIQUE NOT NULL,
    balance         INTEGER NOT NULL DEFAULT 0 CHECK (balance >= 0),
    global_level    INTEGER NOT NULL DEFAULT 1,
    global_xp       INTEGER NOT NULL DEFAULT 0,
    city_id         BIGINT,
    active_job      TEXT,
    corporation_id  BIGINT,
    corporation_role TEXT,
    vip_until       TIMESTAMPTZ,
    daily_streak    INTEGER NOT NULL DEFAULT 0,
    last_daily_at   TIMESTAMPTZ,
    last_active_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_city ON users (city_id);
CREATE INDEX idx_users_corporation ON users (corporation_id);

-- User skills
CREATE TABLE IF NOT EXISTS user_skills (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    direction  TEXT NOT NULL,
    xp         INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, direction)
);

-- Unique constraint for active work: one active work run per user
-- Enforced via partial index (see work_runs migration)
