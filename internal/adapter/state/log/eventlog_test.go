package log

import (
	"context"
	"testing"
	"time"

	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
)

type mockEventLog struct {
	entries map[domlog.ID]domlog.Entry
}

func (m *mockEventLog) AddEntry(_ context.Context, e domlog.Entry) (domlog.ID, error) {
	id := domlog.ID(len(m.entries) + 1)
	m.entries[id] = e
	return id, nil
}

func (m *mockEventLog) ListEntries(_ context.Context, _ ListParams) (EntryList, error) {
	entries := make([]domlog.Entry, 0, len(m.entries))
	for _, e := range m.entries {
		entries = append(entries, e)
	}
	return EntryList{Entries: entries, Total: len(entries)}, nil
}

func TestEventLogInterface(t *testing.T) {
	var el EventLog = &mockEventLog{entries: make(map[domlog.ID]domlog.Entry)}
	e := domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionWork,
		Payload:   map[string]any{"key": "value"},
		Timestamp: time.Now(),
	}
	id, err := el.AddEntry(context.Background(), e)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}
	list, err := el.ListEntries(context.Background(), ListParams{})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if list.Total != 1 {
		t.Errorf("Total = %d, want 1", list.Total)
	}
}

func TestErrNotFound(t *testing.T) {
	if ErrNotFound == nil {
		t.Error("ErrNotFound is nil")
	}
}
