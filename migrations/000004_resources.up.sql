-- Resource definitions (10 base resources, seed-loaded)
CREATE TABLE IF NOT EXISTS resources (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    base_price INTEGER NOT NULL
);
