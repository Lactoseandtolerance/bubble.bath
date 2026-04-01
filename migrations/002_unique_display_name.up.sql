ALTER TABLE users ADD CONSTRAINT chk_display_name_length
CHECK (char_length(display_name) <= 32);

CREATE UNIQUE INDEX idx_users_display_name_unique
ON users (display_name)
WHERE display_name != '';
