package http

import (
	"fmt"
	"time"

	oapi "github.com/4sp1/vipo-go/internal/port/oapi"

	"github.com/4sp1/vipo-go/internal/domain/state/log"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

func PomodoroStateToDomain(s oapi.PomodoroState) (pomodoro.State, error) {
	switch s {
	case oapi.Work:
		return pomodoro.Work, nil
	case oapi.ShortBreak:
		return pomodoro.ShortBreak, nil
	case oapi.LongBreak:
		return pomodoro.LongBreak, nil
	default:
		return pomodoro.State(0), fmt.Errorf("unknown pomodoro state: %s", s)
	}
}

func PomodoroStateFromDomain(s pomodoro.State) (oapi.PomodoroState, error) {
	switch s {
	case pomodoro.Work:
		return oapi.Work, nil
	case pomodoro.ShortBreak:
		return oapi.ShortBreak, nil
	case pomodoro.LongBreak:
		return oapi.LongBreak, nil
	default:
		return "", fmt.Errorf("unknown pomodoro state: %d", s)
	}
}

func LogActionToDomain(a oapi.LogAction) (log.Action, error) {
	switch a {
	case oapi.Start:
		return log.ActionStart, nil
	case oapi.Pause:
		return log.ActionPause, nil
	case oapi.Reset:
		return log.ActionReset, nil
	case oapi.Expire:
		return log.ActionExpire, nil
	case oapi.Resume:
		return log.ActionResume, nil
	case oapi.Select:
		return log.ActionSelect, nil
	default:
		return log.ActionUnknown, fmt.Errorf("unknown log action: %s", a)
	}
}

func LogActionFromDomain(a log.Action) (oapi.LogAction, error) {
	switch a {
	case log.ActionStart:
		return oapi.Start, nil
	case log.ActionPause:
		return oapi.Pause, nil
	case log.ActionReset:
		return oapi.Reset, nil
	case log.ActionExpire:
		return oapi.Expire, nil
	case log.ActionResume:
		return oapi.Resume, nil
	case log.ActionSelect:
		return oapi.Select, nil
	default:
		return "", fmt.Errorf("unknown log action: %d", a)
	}
}

func SessionToDomain(s oapi.PomodoroState) (log.Session, error) {
	switch s {
	case oapi.Work:
		return log.SessionWork, nil
	case oapi.ShortBreak:
		return log.SessionShortBreak, nil
	case oapi.LongBreak:
		return log.SessionLongBreak, nil
	default:
		return log.SessionUnknown, fmt.Errorf("unknown session: %s", s)
	}
}

func SessionFromDomain(s log.Session) (oapi.PomodoroState, error) {
	switch s {
	case log.SessionWork:
		return oapi.Work, nil
	case log.SessionShortBreak:
		return oapi.ShortBreak, nil
	case log.SessionLongBreak:
		return oapi.LongBreak, nil
	default:
		return "", fmt.Errorf("unknown session: %d", s)
	}
}

func TimeToRFC3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func TimeFromRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}