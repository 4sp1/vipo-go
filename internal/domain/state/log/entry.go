package log

import "time"

type Entry struct {
	Event     Event
	Message   Message
	Timestamp time.Time
}

type Message interface {
	isMessage()
}

type MessageAnyTimer struct{}

func (m MessageAnyTimer) isMessage()

type MessageNewNote struct {
	Note string
}

func (m MessageNewNote) isMessage()

type Event int

const (
	EventUnknown Event = iota
	EventStartWork
	EventPauseWork
	EventResetWork
	EventTimerWork
	EventResumeWork
	EventSelectWork
	EventStartShortBreak
	EventPauseShortBreak
	EventResetShortBreak
	EventTimerShortBreak
	EventResumeShortBreak
	EventSelectShortBreak
	EventStartLongBreak
	EventPauseLongBreak
	EventResetLongBreak
	EventTimerLongBreak
	EventResumeLongBreak
	EventSelectLongBreak
	EventNewNote
)
