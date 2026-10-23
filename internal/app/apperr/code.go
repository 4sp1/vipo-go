package apperr

import "fmt"

type Code int

const (
	CodeNoteNotFound Code = iota
	CodeLogNotFound
	CodeInvalidRequest
	CodeInvalidPomodoroState
	CodeInvalidLogAction
	CodeInvalidLimit
	CodeInvalidOffset
	CodeInvalidTimestamp
)

func (c Code) String() string {
	switch c {
	case CodeNoteNotFound:
		return "note_not_found"
	case CodeLogNotFound:
		return "log_not_found"
	case CodeInvalidRequest:
		return "invalid_request"
	case CodeInvalidPomodoroState:
		return "invalid_pomodoro_state"
	case CodeInvalidLogAction:
		return "invalid_log_action"
	case CodeInvalidLimit:
		return "invalid_limit"
	case CodeInvalidOffset:
		return "invalid_offset"
	case CodeInvalidTimestamp:
		return "invalid_timestamp"
	default:
		return fmt.Sprintf("unknown_code(%d)", c)
	}
}

func (c Code) Error() string {
	return c.String()
}
