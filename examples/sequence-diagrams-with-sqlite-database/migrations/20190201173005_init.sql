-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE users
(
  id       SERIAL PRIMARY KEY,
  username TEXT,
  is_contactable BOOL DEFAULT false
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
