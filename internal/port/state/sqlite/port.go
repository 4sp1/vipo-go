package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/4sp1/vipo-go/internal/adapter/state"
	"github.com/4sp1/vipo-go/internal/domain/state/note"

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
		n.Note, n.Pomodoro, n.CreatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert note: %w", err)
	}
	return nil
}

func (p *port) ListNotes(ctx context.Context, params state.ListNotesParams) (state.NoteList, error) {
	baseQuery := "FROM notes"
	whereClause := ""
	args := []any{}

	if params.PomodoroState != nil {
		whereClause = " WHERE pomodoro_state = ?"
		args = append(args, *params.PomodoroState)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery + whereClause
	err := p.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return state.NoteList{}, fmt.Errorf("count notes: %w", err)
	}

	// Get paginated results
	query := "SELECT id, note, pomodoro_state, created_at " + baseQuery + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return state.NoteList{}, fmt.Errorf("query notes: %w", err)
	}
	defer rows.Close()

	var notes []note.State
	for rows.Next() {
		var n note.State
		var createdAtUnix int64
		err := rows.Scan(&n.ID, &n.Note, &n.Pomodoro, &createdAtUnix)
		if err != nil {
			return state.NoteList{}, fmt.Errorf("scan note: %w", err)
		}
		n.CreatedAt = time.Unix(createdAtUnix, 0)
		notes = append(notes, n)
	}

	if notes == nil {
		notes = []note.State{}
	}

	return state.NoteList{
		Notes: notes,
		Total: total,
	}, rows.Err()
}

func (p *port) GetNote(ctx context.Context, id note.ID) (note.State, error) {
	var n note.State
	var createdAtStr string

	err := p.db.QueryRowContext(ctx,
		"SELECT id, note, pomodoro_state, created_at FROM notes WHERE id = ?",
		id,
	).Scan(&n.ID, &n.Note, &n.Pomodoro, &createdAtStr)

	if err != nil {
		if err == sql.ErrNoRows {
			return note.State{}, fmt.Errorf("note not found: %d", id)
		}
		return note.State{}, fmt.Errorf("get note: %w", err)
	}

	n.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return note.State{}, fmt.Errorf("parse created_at: %w", err)
	}

	return n, nil
}

func (p *port) DeleteNote(ctx context.Context, id note.ID) error {
	result, err := p.db.ExecContext(ctx, "DELETE FROM notes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("note not found: %d", id)
	}

	return nil
}
