BEGIN;

ALTER TABLE auth.users ADD COLUMN IF NOT EXISTS registration_type VARCHAR(255);
UPDATE auth.users SET registration_type = 'internal' WHERE registration_type IS NULL;

ALTER TABLE auth.users
ALTER COLUMN registration_type SET NOT NULL,
ALTER COLUMN registration_type SET DEFAULT 'internal';

END;
