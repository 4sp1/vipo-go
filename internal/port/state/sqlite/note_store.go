package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
)

type noteStore struct {
	db *sql.DB
}

func (s *noteStore) AddNote(ctx context.Context, n domnote.State) (domnote.ID, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO notes (note, pomodoro_state, created_at) VALUES (?, ?, ?)",
		n.Note, int(n.Pomodoro), n.CreatedAt.Unix(),
	)
	if err != nil {
		return 0, fmt.Errorf("insert note: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return domnote.ID(id), nil
}

func (s *noteStore) ListNotes(ctx context.Context, params adaptNote.ListParams) (adaptNote.List, error) {
	whereClause := ""
	args := []any{}

	if params.PomodoroState != nil {
		whereClause = " WHERE pomodoro_state = ?"
		args = append(args, int(*params.PomodoroState))
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM notes" + whereClause
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return adaptNote.List{}, fmt.Errorf("count notes: %w", err)
	}

	query := "SELECT id, note, pomodoro_state, created_at FROM notes" + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return adaptNote.List{}, fmt.Errorf("query notes: %w", err)
	}
	defer rows.Close()

	notes := make([]domnote.State, 0)
	for rows.Next() {
		var n domnote.State
		var pomodoroRaw, createdAtUnix int64
		if err := rows.Scan(&n.ID, &n.Note, &pomodoroRaw, &createdAtUnix); err != nil {
			return adaptNote.List{}, fmt.Errorf("scan note: %w", err)
		}
		ps, err := PomodoroStateFromStorage(pomodoroRaw)
		if err != nil {
			return adaptNote.List{}, fmt.Errorf("convert pomodoro state: %w", err)
		}
		n.Pomodoro = ps
		n.CreatedAt = time.Unix(createdAtUnix, 0)
		notes = append(notes, n)
	}

	if err := rows.Err(); err != nil {
		return adaptNote.List{}, fmt.Errorf("rows: %w", err)
	}

	return adaptNote.List{
		Notes: notes,
		Total: total,
	}, nil
}

func (s *noteStore) GetNote(ctx context.Context, id domnote.ID) (domnote.State, error) {
	var n domnote.State
	var pomodoroRaw, createdAtUnix int64

	err := s.db.QueryRowContext(ctx,
		"SELECT id, note, pomodoro_state, created_at FROM notes WHERE id = ?",
		id,
	).Scan(&n.ID, &n.Note, &pomodoroRaw, &createdAtUnix)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domnote.State{}, adaptNote.ErrNotFound
		}
		return domnote.State{}, fmt.Errorf("get note: %w", err)
	}

	ps, err := PomodoroStateFromStorage(pomodoroRaw)
	if err != nil {
		return domnote.State{}, fmt.Errorf("convert pomodoro state: %w", err)
	}
	n.Pomodoro = ps
	n.CreatedAt = time.Unix(createdAtUnix, 0)

	return n, nil
}

func (s *noteStore) DeleteNote(ctx context.Context, id domnote.ID) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM notes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if affected == 0 {
		return adaptNote.ErrNotFound
	}

	return nil
}
