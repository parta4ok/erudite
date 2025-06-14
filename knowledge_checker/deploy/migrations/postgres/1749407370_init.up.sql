BEGIN;

CREATE SCHEMA IF NOT EXISTS kvs;

CREATE TABLE IF NOT EXISTS kvs.question_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE
);

INSERT INTO kvs.question_types (name)
VALUES
    ('single selection'),
    ('multi selection'),
    ('true or false');

CREATE TABLE IF NOT EXISTS kvs.topics (
    id SERIAL PRIMARY KEY,
    topic_id SERIAL NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL UNIQUE
);

INSERT INTO kvs.topics (topic_id, name)
VALUES
    (1, 'Базы данных'),
    (2, 'Базовые типы в Go'),
    (3, 'Составные типы в Go');

CREATE TABLE IF NOT EXISTS kvs.questions (
    id BIGSERIAL PRIMARY KEY,
    question_id SERIAL NOT NULL UNIQUE,
    question_type_id INT NOT NULL REFERENCES kvs.question_types(id),
    topic_id INT NOT NULL REFERENCES kvs.topics(topic_id),
    subject VARCHAR(255) NOT NULL,
    variants TEXT[] NOT NULL,
    correct_answers TEXT[] NOT NULL,
    usage_count INT NOT NULL DEFAULT 0
);

END;