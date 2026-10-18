-- +goose Up
CREATE TABLE queue_entries
(
    id integer PRIMARY KEY AUTOINCREMENT,
    event_id integer NOT NULL,
    user_id integer NOT NULL,
    username text NOT NULL DEFAULT '',
    joined_at integer NOT NULL DEFAULT (unixepoch()),
    UNIQUE (event_id, user_id)
);

-- +goose Down
DROP TABLE queue_entries;