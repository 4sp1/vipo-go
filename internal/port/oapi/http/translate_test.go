package http

import (
	"testing"
	"time"

	oapi "github.com/4sp1/vipo-go/internal/port/oapi"

	"github.com/4sp1/vipo-go/internal/domain/state/log"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

func TestPomodoroStateToDomain(t *testing.T) {
	tests := []struct {
		input    oapi.PomodoroState
		expected pomodoro.State
	}{
		{oapi.Work, pomodoro.Work},
		{oapi.ShortBreak, pomodoro.ShortBreak},
		{oapi.LongBreak, pomodoro.LongBreak},
	}
	for _, tt := range tests {
		got, err := PomodoroStateToDomain(tt.input)
		if err != nil {
			t.Errorf("PomodoroStateToDomain(%q): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("PomodoroStateToDomain(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestPomodoroStateToDomain_UnknownValue(t *testing.T) {
	_, err := PomodoroStateToDomain("unknown")
	if err == nil {
		t.Error("expected error for unknown PomodoroState, got nil")
	}
}

func TestPomodoroStateFromDomain(t *testing.T) {
	tests := []struct {
		input    pomodoro.State
		expected oapi.PomodoroState
	}{
		{pomodoro.Work, oapi.Work},
		{pomodoro.ShortBreak, oapi.ShortBreak},
		{pomodoro.LongBreak, oapi.LongBreak},
	}
	for _, tt := range tests {
		got, err := PomodoroStateFromDomain(tt.input)
		if err != nil {
			t.Errorf("PomodoroStateFromDomain(%d): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("PomodoroStateFromDomain(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestPomodoroStateFromDomain_UnknownValue(t *testing.T) {
	_, err := PomodoroStateFromDomain(pomodoro.State(99))
	if err == nil {
		t.Error("expected error for unknown pomodoro.State, got nil")
	}
}

func TestLogActionToDomain(t *testing.T) {
	tests := []struct {
		input    oapi.LogAction
		expected log.Action
	}{
		{oapi.Start, log.ActionStart},
		{oapi.Pause, log.ActionPause},
		{oapi.Reset, log.ActionReset},
		{oapi.Expire, log.ActionExpire},
		{oapi.Resume, log.ActionResume},
		{oapi.Select, log.ActionSelect},
	}
	for _, tt := range tests {
		got, err := LogActionToDomain(tt.input)
		if err != nil {
			t.Errorf("LogActionToDomain(%q): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("LogActionToDomain(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestLogActionToDomain_UnknownValue(t *testing.T) {
	_, err := LogActionToDomain("unknown")
	if err == nil {
		t.Error("expected error for unknown LogAction, got nil")
	}
}

func TestLogActionFromDomain(t *testing.T) {
	tests := []struct {
		input    log.Action
		expected oapi.LogAction
	}{
		{log.ActionStart, oapi.Start},
		{log.ActionPause, oapi.Pause},
		{log.ActionReset, oapi.Reset},
		{log.ActionExpire, oapi.Expire},
		{log.ActionResume, oapi.Resume},
		{log.ActionSelect, oapi.Select},
	}
	for _, tt := range tests {
		got, err := LogActionFromDomain(tt.input)
		if err != nil {
			t.Errorf("LogActionFromDomain(%d): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("LogActionFromDomain(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestLogActionFromDomain_UnknownValue(t *testing.T) {
	_, err := LogActionFromDomain(log.ActionUnknown)
	if err == nil {
		t.Error("expected error for ActionUnknown, got nil")
	}
}

func TestSessionToDomain(t *testing.T) {
	tests := []struct {
		input    oapi.PomodoroState
		expected log.Session
	}{
		{oapi.Work, log.SessionWork},
		{oapi.ShortBreak, log.SessionShortBreak},
		{oapi.LongBreak, log.SessionLongBreak},
	}
	for _, tt := range tests {
		got, err := SessionToDomain(tt.input)
		if err != nil {
			t.Errorf("SessionToDomain(%q): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("SessionToDomain(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestSessionToDomain_UnknownValue(t *testing.T) {
	_, err := SessionToDomain("unknown")
	if err == nil {
		t.Error("expected error for unknown session, got nil")
	}
}

func TestSessionFromDomain(t *testing.T) {
	tests := []struct {
		input    log.Session
		expected oapi.PomodoroState
	}{
		{log.SessionWork, oapi.Work},
		{log.SessionShortBreak, oapi.ShortBreak},
		{log.SessionLongBreak, oapi.LongBreak},
	}
	for _, tt := range tests {
		got, err := SessionFromDomain(tt.input)
		if err != nil {
			t.Errorf("SessionFromDomain(%d): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("SessionFromDomain(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSessionFromDomain_UnknownValue(t *testing.T) {
	_, err := SessionFromDomain(log.SessionUnknown)
	if err == nil {
		t.Error("expected error for SessionUnknown, got nil")
	}
}

func TestTimeToRFC3339(t *testing.T) {
	ts := time.Date(2026, 6, 14, 12, 30, 0, 0, time.UTC)
	got := TimeToRFC3339(ts)
	expected := "2026-06-14T12:30:00Z"
	if got != expected {
		t.Errorf("TimeToRFC3339() = %q, want %q", got, expected)
	}
}

func TestTimeToRFC3339_NormalizesToUTC(t *testing.T) {
	loc := time.FixedZone("EST", -5*3600)
	ts := time.Date(2026, 6, 14, 7, 30, 0, 0, loc)
	got := TimeToRFC3339(ts)
	expected := "2026-06-14T12:30:00Z"
	if got != expected {
		t.Errorf("TimeToRFC3339() = %q, want %q", got, expected)
	}
}

func TestTimeFromRFC3339(t *testing.T) {
	input := "2026-06-14T12:30:00Z"
	got, err := TimeFromRFC3339(input)
	if err != nil {
		t.Errorf("TimeFromRFC3339(%q): unexpected error: %v", input, err)
	}
	expected := time.Date(2026, 6, 14, 12, 30, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("TimeFromRFC3339(%q) = %v, want %v", input, got, expected)
	}
}

func TestTimeFromRFC3339_InvalidInput(t *testing.T) {
	_, err := TimeFromRFC3339("not-a-timestamp")
	if err == nil {
		t.Error("expected error for invalid RFC3339 input, got nil")
	}
}

func TestPomodoroStateRoundTrip(t *testing.T) {
	states := []pomodoro.State{pomodoro.Work, pomodoro.ShortBreak, pomodoro.LongBreak}
	for _, original := range states {
		oapiState, err := PomodoroStateFromDomain(original)
		if err != nil {
			t.Errorf("PomodoroStateFromDomain(%d): unexpected error: %v", original, err)
		}
		back, err := PomodoroStateToDomain(oapiState)
		if err != nil {
			t.Errorf("PomodoroStateToDomain(%q): unexpected error: %v", oapiState, err)
		}
		if back != original {
			t.Errorf("round trip: domain %d → oapi %q → domain %d", original, oapiState, back)
		}
	}
}

func TestLogActionRoundTrip(t *testing.T) {
	actions := []log.Action{log.ActionStart, log.ActionPause, log.ActionReset, log.ActionExpire, log.ActionResume, log.ActionSelect}
	for _, original := range actions {
		oapiAction, err := LogActionFromDomain(original)
		if err != nil {
			t.Errorf("LogActionFromDomain(%d): unexpected error: %v", original, err)
		}
		back, err := LogActionToDomain(oapiAction)
		if err != nil {
			t.Errorf("LogActionToDomain(%q): unexpected error: %v", oapiAction, err)
		}
		if back != original {
			t.Errorf("round trip: domain %d → oapi %q → domain %d", original, oapiAction, back)
		}
	}
}

func TestTimeRoundTrip(t *testing.T) {
	original := time.Date(2026, 6, 14, 12, 30, 0, 0, time.UTC)
	rfc3339 := TimeToRFC3339(original)
	back, err := TimeFromRFC3339(rfc3339)
	if err != nil {
		t.Errorf("TimeFromRFC3339(%q): unexpected error: %v", rfc3339, err)
	}
	original = original.Truncate(time.Second)
	if !back.Equal(original) {
		t.Errorf("round trip: %v → %q → %v", original, rfc3339, back)
	}
}