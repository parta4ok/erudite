BEGIN;

CREATE TABLE IF NOT EXISTS kvs.sessions (
    id SERIAL PRIMARY KEY,
    session_id BIGSERIAL,
    user_id SERIAL NOT NULL,
    state VARCHAR(255) NOT NULL, 
    topics TEXT[] NOT NULL,
    questions INTEGER[],
    answers JSON,
    created_at TIMESTAMP,
    duration_limit BIGSERIAL,
    is_expired BOOLEAN,
    is_passed BOOLEAN,
    comment VARCHAR(255)
);

END;