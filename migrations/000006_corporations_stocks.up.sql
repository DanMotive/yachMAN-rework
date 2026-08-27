-- Corporations
CREATE TABLE IF NOT EXISTS corporations (
    id                    BIGSERIAL PRIMARY KEY,
    name                  TEXT NOT NULL UNIQUE,
    owner_user_id         BIGINT NOT NULL REFERENCES users(id),
    balance               INTEGER NOT NULL DEFAULT 5000000,
    registration_fee_paid INTEGER NOT NULL DEFAULT 1000000,
    total_shares          INTEGER NOT NULL DEFAULT 10000000,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Corporation staff
CREATE TABLE IF NOT EXISTS corporation_staff (
    id              BIGSERIAL PRIMARY KEY,
    corporation_id  BIGINT NOT NULL REFERENCES corporations(id) ON DELETE CASCADE,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    position        TEXT NOT NULL,
    salary          INTEGER NOT NULL DEFAULT 0,
    hired_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_corp_staff_unique ON corporation_staff (corporation_id, user_id);

-- Shares
CREATE TABLE IF NOT EXISTS shares (
    id              BIGSERIAL PRIMARY KEY,
    corporation_id  BIGINT NOT NULL REFERENCES corporations(id) ON DELETE CASCADE,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount          INTEGER NOT NULL DEFAULT 0 CHECK (amount >= 0)
);

CREATE UNIQUE INDEX idx_shares_unique ON shares (corporation_id, user_id);

-- Share orders (buy/sell)
CREATE TABLE IF NOT EXISTS share_orders (
    id              BIGSERIAL PRIMARY KEY,
    corporation_id  BIGINT NOT NULL REFERENCES corporations(id) ON DELETE CASCADE,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_type      TEXT NOT NULL CHECK (order_type IN ('buy', 'sell')),
    price_per_share INTEGER NOT NULL,
    amount          INTEGER NOT NULL,
    filled          INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'open',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    operation_id    UUID NOT NULL UNIQUE
);

CREATE INDEX idx_share_orders_open ON share_orders (corporation_id, status)
    WHERE status = 'open';

-- Foreign key: users.corporation_id -> corporations.id
ALTER TABLE users ADD CONSTRAINT fk_users_corporation
    FOREIGN KEY (corporation_id) REFERENCES corporations(id);
