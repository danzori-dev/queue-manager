-- +goose Up
ALTER TABLE queue_entries
    ADD COLUMN sort_key integer;

UPDATE queue_entries
SET sort_key = id;

-- +goose Down
ALTER TABLE queue_entries DROP COLUMN sort_key;