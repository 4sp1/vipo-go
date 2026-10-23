package log

import "time"

type ID int64

type Entry struct {
	ID        ID
	Action    Action
	Session   Session
	Payload   map[string]any
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
)

type Session int

const (
	SessionUnknown Session = iota
	SessionWork
	SessionShortBreak
	SessionLongBreak
)
