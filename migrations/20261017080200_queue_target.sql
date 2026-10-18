-- +goose Up
CREATE TABLE queue_target
(
    id integer PRIMARY KEY CHECK (id = 1),
    chat_id integer NOT NULL,
    thread_id integer NOT NULL
);

-- +goose Down
DROP TABLE queue_target;