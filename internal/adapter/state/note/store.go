package note

import (
	"context"
	"errors"

	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

type ListParams struct {
	PomodoroState *pomodoro.State
	Limit, Offset int
}

type List struct {
	Notes []domnote.State
	Total int
}

type Store interface {
	AddNote(ctx context.Context, n domnote.State) (domnote.ID, error)
	ListNotes(ctx context.Context, params ListParams) (List, error)
	GetNote(ctx context.Context, id domnote.ID) (domnote.State, error)
	DeleteNote(ctx context.Context, id domnote.ID) error
}

var ErrNotFound = errors.New("adapter: note not found")
