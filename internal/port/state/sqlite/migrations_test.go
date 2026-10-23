package sqlite

import (
	"context"
	"testing"
	"time"

	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

func testNote(text string) (n domnote.State) {
	n.Note = text
	n.Pomodoro = pomodoro.Work
	n.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return
}

func testLogEntry() (e domlog.Entry) {
	e.Action = domlog.ActionStart
	e.Session = domlog.SessionWork
	e.Payload = map[string]any{"test": true}
	e.Timestamp = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return
}

func TestMigrateCreatesTables(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	ns := db.NoteStore()
	id, err := ns.AddNote(context.Background(), testNote("migration test"))
	if err != nil {
		t.Fatalf("AddNote after migrate: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}

	el := db.EventLog()
	logID, err := el.AddEntry(context.Background(), testLogEntry())
	if err != nil {
		t.Fatalf("AddEntry after migrate: %v", err)
	}
	if logID != 1 {
		t.Errorf("log id = %d, want 1", logID)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate 1: %v", err)
	}
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate 2: %v", err)
	}
}
