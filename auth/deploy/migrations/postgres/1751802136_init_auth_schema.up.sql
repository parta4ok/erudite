BEGIN;


CREATE SCHEMA IF NOT EXISTS auth;


CREATE TABLE IF NOT EXISTS auth.users (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    name VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    rights TEXT[] NOT NULL DEFAULT '{}',
    contacts JSON,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
    );


END;