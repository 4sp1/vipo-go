package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/vipo-org/vipo-server/internal/adapter/state"
	"github.com/vipo-org/vipo-server/internal/domain/state/note"

	_ "github.com/glebarez/go-sqlite"
)

func main(file string) (state.Repository, error) {
	db, err := sql.Open("sqlite", file)
	if err != nil {
		return nil, fmt.Errorf("sql open: %w", err)
	}
	return &port{
		db: db,
	}, nil
}

type port struct {
	db *sql.DB
}

func (p *port) AddNote(ctx context.Context, note note.State) error
