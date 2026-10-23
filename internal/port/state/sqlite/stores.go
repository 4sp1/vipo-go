package sqlite

import "database/sql"

type noteStore struct {
	db *sql.DB
}

type eventLog struct {
	db *sql.DB
}
