package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/glebarez/go-sqlite"
)

type DB struct {
	db *sql.DB
}

func New(file string) (*DB, error) {
	db, err := sql.Open("sqlite", file)
	if err != nil {
		return nil, fmt.Errorf("sql open: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) NoteStore() *noteStore {
	return &noteStore{db: d.db}
}

func (d *DB) EventLog() *eventLog {
	return &eventLog{db: d.db}
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) PingContext(ctx context.Context) error {
	return d.db.PingContext(ctx)
}
