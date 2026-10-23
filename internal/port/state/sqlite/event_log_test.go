package sqlite

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
)

func setupLogTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = db.db.Exec(`CREATE TABLE log_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		action INTEGER NOT NULL,
		session INTEGER NOT NULL,
		payload TEXT NOT NULL,
		timestamp INTEGER NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestEventLogAddEntryReturnsID(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	entry := domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionWork,
		Payload:   map[string]any{"key": "value"},
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	id, err := el.AddEntry(context.Background(), entry)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}
}

func TestEventLogAddEntryNilPayload(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	entry := domlog.Entry{
		Action:    domlog.ActionPause,
		Session:   domlog.SessionShortBreak,
		Payload:   nil,
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	_, err := el.AddEntry(context.Background(), entry)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if list.Total != 1 {
		t.Fatalf("Total = %d, want 1", list.Total)
	}
	if list.Entries[0].Payload != nil {
		t.Errorf("Payload = %v, want nil", list.Entries[0].Payload)
	}
}

func TestEventLogListEntries(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	for i := 0; i < 5; i++ {
		entry := domlog.Entry{
			Action:    domlog.ActionStart,
			Session:   domlog.SessionWork,
			Payload:   map[string]any{"index": i},
			Timestamp: time.Date(2026, 1, 1, 0, i, 0, 0, time.UTC),
		}
		_, err := el.AddEntry(context.Background(), entry)
		if err != nil {
			t.Fatalf("AddEntry %d: %v", i, err)
		}
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(list.Entries) != 3 {
		t.Errorf("len(Entries) = %d, want 3", len(list.Entries))
	}
	if list.Total != 5 {
		t.Errorf("Total = %d, want 5", list.Total)
	}
}

func TestEventLogListEntriesFilterByAction(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	entries := []struct {
		action  domlog.Action
		session domlog.Session
	}{
		{domlog.ActionStart, domlog.SessionWork},
		{domlog.ActionPause, domlog.SessionWork},
		{domlog.ActionStart, domlog.SessionShortBreak},
	}
	for _, e := range entries {
		_, err := el.AddEntry(context.Background(), domlog.Entry{
			Action:    e.action,
			Session:   e.session,
			Payload:   map[string]any{},
			Timestamp: time.Now(),
		})
		if err != nil {
			t.Fatalf("AddEntry: %v", err)
		}
	}

	action := domlog.ActionStart
	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Action: &action, Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(list.Entries) != 2 {
		t.Errorf("len(Entries) = %d, want 2", len(list.Entries))
	}
	if list.Total != 2 {
		t.Errorf("Total = %d, want 2", list.Total)
	}
}

func TestEventLogListEntriesFilterBySession(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	session := domlog.SessionWork
	_, err := el.AddEntry(context.Background(), domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionWork,
		Payload:   map[string]any{},
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionShortBreak,
		Payload:   map[string]any{},
		Timestamp: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Session: &session, Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(list.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(list.Entries))
	}
	if list.Total != 1 {
		t.Errorf("Total = %d, want 1", list.Total)
	}
}

func TestEventLogListEntriesFilterByTimeRange(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	from := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC)

	_, err := el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionStart, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionStart, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 0, 1, 30, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionStart, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 0, 3, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{From: &from, To: &to, Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(list.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(list.Entries))
	}
	if list.Total != 1 {
		t.Errorf("Total = %d, want 1", list.Total)
	}
}

func TestEventLogListEntriesEmpty(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if list.Entries == nil {
		t.Error("Entries is nil, want empty slice")
	}
	if len(list.Entries) != 0 {
		t.Errorf("len(Entries) = %d, want 0", len(list.Entries))
	}
	if list.Total != 0 {
		t.Errorf("Total = %d, want 0", list.Total)
	}
}

func TestEventLogPayloadRoundTrip(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	original := map[string]any{
		"string": "hello",
		"number": float64(42),
		"nested": map[string]any{"inner": "value"},
		"array":  []any{1, 2, 3},
	}
	entry := domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionWork,
		Payload:   original,
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	id, err := el.AddEntry(context.Background(), entry)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}

	got := list.Entries[0]
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}

	originalJSON, _ := json.Marshal(original)
	gotJSON, _ := json.Marshal(got.Payload)
	if string(originalJSON) != string(gotJSON) {
		t.Errorf("payload mismatch:\n  orig: %s\n  got:  %s", originalJSON, gotJSON)
	}
}

func TestEventLogListEntriesFiltersCombined(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	action := domlog.ActionPause
	session := domlog.SessionWork
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 23, 59, 59, 0, time.UTC)

	_, err := el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionPause, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionStart, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionPause, Session: domlog.SessionShortBreak,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionPause, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{
		Action:  &action,
		Session: &session,
		From:    &from,
		To:      &to,
		Limit:   50,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(list.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(list.Entries))
	}
	if list.Total != 1 {
		t.Errorf("Total = %d, want 1", list.Total)
	}
}
