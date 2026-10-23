package svc

import "fmt"

type ErrInvalidPomodoroState struct{ Value string }

func (e *ErrInvalidPomodoroState) Error() string {
	return fmt.Sprintf("invalid pomodoro_state: %s", e.Value)
}

type ErrInvalidLogAction struct{ Value string }

func (e *ErrInvalidLogAction) Error() string {
	return fmt.Sprintf("invalid action: %s", e.Value)
}

type ErrInvalidSession struct{ Value string }

func (e *ErrInvalidSession) Error() string {
	return fmt.Sprintf("invalid session: %s", e.Value)
}

type ErrInvalidLimit struct{ Value int }

func (e *ErrInvalidLimit) Error() string {
	return fmt.Sprintf("invalid limit: %d", e.Value)
}

type ErrInvalidOffset struct{ Value int }

func (e *ErrInvalidOffset) Error() string {
	return fmt.Sprintf("invalid offset: %d", e.Value)
}

type ErrInvalidRequest struct{ Message string }

func (e *ErrInvalidRequest) Error() string {
	if e.Message == "" {
		return "invalid request"
	}
	return e.Message
}
