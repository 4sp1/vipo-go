package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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

func (p *port) AddNote(ctx context.Context, n note.State) error {
	_, err := p.db.ExecContext(ctx,
		"INSERT INTO notes (note, pomodoro_state, created_at) VALUES (?, ?, ?)",
		n.Note, n.Pomodoro, n.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert note: %w", err)
	}
	return nil
}
