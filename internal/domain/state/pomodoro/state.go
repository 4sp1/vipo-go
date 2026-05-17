package pomodoro

type State int

const (
	Work State = iota
	ShortBreak
	LongBreak
)
