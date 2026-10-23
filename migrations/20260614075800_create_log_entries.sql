-- +goose Up
CREATE TABLE log_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action INTEGER NOT NULL,
    session INTEGER NOT NULL,
    payload TEXT NOT NULL,
    timestamp INTEGER NOT NULL
);

-- +goose Down
DROP TABLE log_entries;
