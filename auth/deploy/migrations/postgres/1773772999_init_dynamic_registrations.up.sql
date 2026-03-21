BEGIN;

CREATE TABLE IF NOT EXISTS auth.dynamic_registrations (
    id         SERIAL PRIMARY KEY,
    sid        TEXT NOT NULL,
    code       INTEGER NOT NULL,
    provider   VARCHAR(255),
    contact    VARCHAR(255),
    created_at TIMESTAMP NOT NULL,
    approve_period INTERVAL NOT NULL DEFAULT '2 minutes'
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sid ON auth.dynamic_registrations (sid);

END;
