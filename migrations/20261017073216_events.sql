-- +goose Up
CREATE TABLE events
(
    id integer PRIMARY KEY AUTOINCREMENT,
    chat_id integer NOT NULL,
    thread_id integer NOT NULL DEFAULT 0,
    message_id integer,
    title text NOT NULL,
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at integer NOT NULL DEFAULT (unixepoch()),
    closed_at integer
);

-- +goose Down
DROP TABLE events;