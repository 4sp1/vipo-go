package sqlite

import (
	"context"
	"testing"
)

func TestNewOpensDatabase(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestNoteStoreReturnsImplementor(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	ns := db.NoteStore()
	if ns == nil {
		t.Error("NoteStore returned nil")
	}
}

func TestEventLogReturnsImplementor(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	el := db.EventLog()
	if el == nil {
		t.Error("EventLog returned nil")
	}
}
