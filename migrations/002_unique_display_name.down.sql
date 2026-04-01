DROP INDEX IF EXISTS idx_users_display_name_unique;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_display_name_length;
