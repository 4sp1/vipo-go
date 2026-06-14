package log

import "time"

type Entry struct {
	Action    Action
	Session   Session
	Payload   Message
	Timestamp time.Time
}

type Action int

const (
	ActionUnknown Action = iota
	ActionStart
	ActionPause
	ActionReset
	ActionExpire
	ActionResume
	ActionSelect
	ActionNewNote
)

type Session int

const (
	SessionUnknown Session = iota
	SessionWork
	SessionShortBreak
	SessionLongBreak
)

type Message interface {
	isMessage()
}

type MessageNewNote struct {
	Note string
}

func (m MessageNewNote) isMessage() {}