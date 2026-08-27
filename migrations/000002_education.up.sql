-- Education programs (seed-loaded)
CREATE TABLE IF NOT EXISTS education_programs (
    id                   TEXT PRIMARY KEY,
    name                 TEXT NOT NULL,
    direction            TEXT NOT NULL,
    required_xp          INTEGER NOT NULL,
    cost                 INTEGER NOT NULL,
    lesson_count         INTEGER NOT NULL,
    lesson_interval_hours INTEGER NOT NULL DEFAULT 12
);

-- User education progress
CREATE TABLE IF NOT EXISTS user_education (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    program_id      TEXT NOT NULL REFERENCES education_programs(id),
    progress        INTEGER NOT NULL DEFAULT 0,
    completed       BOOLEAN NOT NULL DEFAULT FALSE,
    next_lesson_at  TIMESTAMPTZ,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

-- Only one active (non-completed) enrollment per user per program
CREATE UNIQUE INDEX idx_user_education_active ON user_education (user_id, program_id)
    WHERE completed = FALSE;
