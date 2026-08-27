-- Trade contracts between cities
CREATE TABLE IF NOT EXISTS trade_contracts (
    id               BIGSERIAL PRIMARY KEY,
    from_city_id     BIGINT NOT NULL REFERENCES cities(id),
    to_city_id       BIGINT NOT NULL REFERENCES cities(id),
    resource_id      TEXT NOT NULL REFERENCES resources(id),
    quantity_per_day INTEGER NOT NULL CHECK (quantity_per_day >= 10),
    price_per_unit   INTEGER NOT NULL,
    payers           TEXT NOT NULL CHECK (payers IN ('from', 'to')),
    duration_days    INTEGER NOT NULL,
    start_at         TIMESTAMPTZ,
    penalty_pct      INTEGER NOT NULL DEFAULT 10,
    status           TEXT NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    operation_id     UUID NOT NULL UNIQUE
);

-- Ledger entries for all financial operations
CREATE TABLE IF NOT EXISTS ledger_entries (
    id            BIGSERIAL PRIMARY KEY,
    operation_id  UUID NOT NULL UNIQUE,
    entity_type   TEXT NOT NULL CHECK (entity_type IN ('user', 'city', 'corporation')),
    entity_id     BIGINT NOT NULL,
    debit         INTEGER NOT NULL DEFAULT 0,
    credit        INTEGER NOT NULL DEFAULT 0,
    balance_after INTEGER NOT NULL,
    description   TEXT DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ledger_entity ON ledger_entries (entity_type, entity_id, created_at);
