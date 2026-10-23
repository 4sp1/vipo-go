package log

import (
	"context"
	"errors"
	"time"

	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
)

type ListParams struct {
	Action        *domlog.Action
	Session       *domlog.Session
	From, To      *time.Time
	Limit, Offset int
}

type EntryList struct {
	Entries []domlog.Entry
	Total   int
}

type EventLog interface {
	AddEntry(ctx context.Context, entry domlog.Entry) (domlog.ID, error)
	ListEntries(ctx context.Context, params ListParams) (EntryList, error)
}

var ErrNotFound = errors.New("adapter: log entry not found")
