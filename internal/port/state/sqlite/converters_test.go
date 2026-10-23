package sqlite

import (
	"testing"

	"github.com/4sp1/vipo-go/internal/domain/state/log"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

func TestPomodoroStateFromStorage(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		want    pomodoro.State
		wantErr bool
	}{
		{"work", 0, pomodoro.Work, false},
		{"short_break", 1, pomodoro.ShortBreak, false},
		{"long_break", 2, pomodoro.LongBreak, false},
		{"negative", -1, pomodoro.State(0), true},
		{"out_of_range", 3, pomodoro.State(0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PomodoroStateFromStorage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLogActionFromStorage(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		want    log.Action
		wantErr bool
	}{
		{"unknown", 0, log.Action(0), true},
		{"start", 1, log.ActionStart, false},
		{"pause", 2, log.ActionPause, false},
		{"reset", 3, log.ActionReset, false},
		{"expire", 4, log.ActionExpire, false},
		{"resume", 5, log.ActionResume, false},
		{"select", 6, log.ActionSelect, false},
		{"new_note_removed", 7, log.Action(0), true},
		{"negative", -1, log.Action(0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LogActionFromStorage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLogSessionFromStorage(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		want    log.Session
		wantErr bool
	}{
		{"unknown", 0, log.Session(0), true},
		{"work", 1, log.SessionWork, false},
		{"short_break", 2, log.SessionShortBreak, false},
		{"long_break", 3, log.SessionLongBreak, false},
		{"out_of_range", 4, log.Session(0), true},
		{"negative", -1, log.Session(0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LogSessionFromStorage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got = %d, want %d", got, tt.want)
			}
		})
	}
}
