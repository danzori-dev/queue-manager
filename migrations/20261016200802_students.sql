-- +goose Up
CREATE TABLE students
(
    user_id integer PRIMARY KEY,
    name text NOT NULL
);

-- +goose Down
DROP TABLE students;