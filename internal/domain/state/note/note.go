package note

import (
	"time"

	"github.com/vipo-org/vipo-server/internal/domain/state/pomodoro"
)

type ID int64

type State struct {
	ID        ID
	Note      string
	Pomodoro  pomodoro.State
	CreatedAt time.Time
}
