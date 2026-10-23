package apperr

import (
	"testing"
)

func TestCodeString(t *testing.T) {
	tests := []struct {
		code Code
		want string
	}{
		{CodeNoteNotFound, "note_not_found"},
		{CodeLogNotFound, "log_not_found"},
		{CodeInvalidRequest, "invalid_request"},
		{CodeInvalidPomodoroState, "invalid_pomodoro_state"},
		{CodeInvalidLogAction, "invalid_log_action"},
		{CodeInvalidLimit, "invalid_limit"},
		{CodeInvalidOffset, "invalid_offset"},
		{CodeInvalidTimestamp, "invalid_timestamp"},
	}
	for _, tt := range tests {
		if got := tt.code.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestCodeError(t *testing.T) {
	c := CodeNoteNotFound
	if c.Error() != "note_not_found" {
		t.Errorf("Error() = %q, want 'note_not_found'", c.Error())
	}
}
