-- +goose Up
-- Explicit per-user interface-language override. Empty means "derive from the
-- Telegram profile" (users.language_code); a non-empty value such as 'rofl' wins.
-- Kept separate from language_code (Telegram raw hint), so Upsert can refresh the
-- hint without clobbering the override.
ALTER TABLE users ADD COLUMN ui_language text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN ui_language;
