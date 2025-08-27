BEGIN;

DROP INDEX IF EXISTS auth.idx_users_group_id;

ALTER TABLE auth.users DROP CONSTRAINT IF EXISTS fk_users_group_id;

ALTER TABLE auth.users RENAME COLUMN group_id TO linked_id;

END;