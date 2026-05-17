package note

import (
	"time"

	"github.com/vipo-org/vipo-server/internal/domain/state/pomodoro"
)

type State struct {
	Note      string
	Pomodoro  pomodoro.State
	CreatedAt time.Time
}
