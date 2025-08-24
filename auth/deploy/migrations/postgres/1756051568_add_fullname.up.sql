BEGIN;

ALTER TABLE auth.users ADD COLUMN IF NOT EXISTS fullname TEXT;

UPDATE auth.users SET fullname = name WHERE fullname IS NULL;

ALTER TABLE auth.users ALTER COLUMN fullname SET NOT NULL;

END;