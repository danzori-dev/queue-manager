-- +goose Up
CREATE TABLE admins
(
    user_id integer PRIMARY KEY,
    created_at integer NOT NULL DEFAULT (unixepoch())
);

-- +goose Down
DROP TABLE admins;