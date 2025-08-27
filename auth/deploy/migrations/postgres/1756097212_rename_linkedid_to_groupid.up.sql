BEGIN;

ALTER TABLE auth.users RENAME COLUMN linked_id TO group_id;

UPDATE auth.users SET group_id = NULL WHERE group_id = '';

ALTER TABLE auth.users 
ADD CONSTRAINT fk_users_group_id 
FOREIGN KEY (group_id) REFERENCES auth.groups(gid)
ON UPDATE CASCADE 
ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_users_group_id ON auth.users(group_id);

END;