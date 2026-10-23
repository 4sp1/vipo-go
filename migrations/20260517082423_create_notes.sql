-- +goose Up
CREATE TABLE notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note TEXT NOT NULL,
    pomodoro_state INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE notes;
