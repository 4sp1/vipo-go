package state

import (
	"context"

	"github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

type ListNotesParams struct {
	PomodoroState *pomodoro.State
	Limit         int
	Offset        int
}

type NoteList struct {
	Notes []note.State
	Total int
}

type Repository interface {
	AddNote(ctx context.Context, note note.State) error
	ListNotes(ctx context.Context, params ListNotesParams) (NoteList, error)
	GetNote(ctx context.Context, id note.ID) (note.State, error)
	DeleteNote(ctx context.Context, id note.ID) error
}
