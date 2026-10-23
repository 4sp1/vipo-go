package log

import (
	"testing"
	"time"
)

func TestEntryConstruction(t *testing.T) {
	entry := Entry{
		ID:        ID(42),
		Action:    ActionStart,
		Session:   SessionWork,
		Payload:   map[string]any{"key": "value"},
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if entry.ID != 42 {
		t.Errorf("ID = %d, want 42", entry.ID)
	}
	if entry.Action != ActionStart {
		t.Errorf("Action = %d, want %d", entry.Action, ActionStart)
	}
	if entry.Session != SessionWork {
		t.Errorf("Session = %d, want %d", entry.Session, SessionWork)
	}
	if entry.Payload["key"] != "value" {
		t.Errorf("Payload[key] = %v, want value", entry.Payload["key"])
	}
}

func TestActionValues(t *testing.T) {
	tests := []struct {
		name  string
		value Action
		want  int
	}{
		{"ActionUnknown", ActionUnknown, 0},
		{"ActionStart", ActionStart, 1},
		{"ActionPause", ActionPause, 2},
		{"ActionReset", ActionReset, 3},
		{"ActionExpire", ActionExpire, 4},
		{"ActionResume", ActionResume, 5},
		{"ActionSelect", ActionSelect, 6},
	}
	for _, tt := range tests {
		if int(tt.value) != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

func TestSessionValues(t *testing.T) {
	tests := []struct {
		name  string
		value Session
		want  int
	}{
		{"SessionUnknown", SessionUnknown, 0},
		{"SessionWork", SessionWork, 1},
		{"SessionShortBreak", SessionShortBreak, 2},
		{"SessionLongBreak", SessionLongBreak, 3},
	}
	for _, tt := range tests {
		if int(tt.value) != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}
