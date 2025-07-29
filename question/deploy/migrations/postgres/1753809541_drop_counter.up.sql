BEGIN;

ALTER TABLE kvs.questions DROP COLUMN IF EXISTS usage_count;

END;