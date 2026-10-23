package sqlite

import (
	"fmt"

	"github.com/4sp1/vipo-go/internal/domain/state/log"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

func PomodoroStateFromStorage(v int64) (pomodoro.State, error) {
	switch v {
	case 0:
		return pomodoro.Work, nil
	case 1:
		return pomodoro.ShortBreak, nil
	case 2:
		return pomodoro.LongBreak, nil
	default:
		return pomodoro.State(0), fmt.Errorf("invalid pomodoro state from storage: %d", v)
	}
}

func LogActionFromStorage(v int64) (log.Action, error) {
	switch log.Action(v) {
	case log.ActionStart:
		return log.ActionStart, nil
	case log.ActionPause:
		return log.ActionPause, nil
	case log.ActionReset:
		return log.ActionReset, nil
	case log.ActionExpire:
		return log.ActionExpire, nil
	case log.ActionResume:
		return log.ActionResume, nil
	case log.ActionSelect:
		return log.ActionSelect, nil
	default:
		return log.Action(0), fmt.Errorf("invalid log action from storage: %d", v)
	}
}

func LogSessionFromStorage(v int64) (log.Session, error) {
	switch log.Session(v) {
	case log.SessionWork:
		return log.SessionWork, nil
	case log.SessionShortBreak:
		return log.SessionShortBreak, nil
	case log.SessionLongBreak:
		return log.SessionLongBreak, nil
	default:
		return log.Session(0), fmt.Errorf("invalid log session from storage: %d", v)
	}
}
