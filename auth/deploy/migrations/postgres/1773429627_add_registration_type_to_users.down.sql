BEGIN;

ALTER TABLE auth.users DROP COLUMN IF EXISTS registration_type;

END;
