package state

import (
	"context"

	"github.com/vipo-org/vipo-server/internal/domain/state/note"
)

type Repository interface {
	AddNote(ctx context.Context, note note.State) error
}
