-- Work definitions (seed-loaded, 300 entries)
CREATE TABLE IF NOT EXISTS work_definitions (
    id                 TEXT PRIMARY KEY,
    name               TEXT NOT NULL,
    direction          TEXT NOT NULL,
    required_xp        INTEGER NOT NULL DEFAULT 0,
    duration_minutes   INTEGER NOT NULL,
    payout             INTEGER NOT NULL,
    xp_reward          INTEGER NOT NULL,
    resource_type      TEXT NOT NULL,
    resource_amount    INTEGER NOT NULL
);

-- Active work runs (timers)
CREATE TABLE IF NOT EXISTS work_runs (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    work_id      TEXT NOT NULL REFERENCES work_definitions(id),
    city_id      BIGINT NOT NULL,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finishes_at  TIMESTAMPTZ NOT NULL,
    completed    BOOLEAN NOT NULL DEFAULT FALSE,
    operation_id UUID NOT NULL UNIQUE
);

-- Enforce one active (uncompleted) work run per user
CREATE UNIQUE INDEX idx_work_runs_active ON work_runs (user_id)
    WHERE completed = FALSE;

CREATE INDEX idx_work_runs_finishes ON work_runs (finishes_at)
    WHERE completed = FALSE;
