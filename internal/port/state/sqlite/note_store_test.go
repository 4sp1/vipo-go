package sqlite

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = db.db.Exec(`CREATE TABLE notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		note TEXT NOT NULL,
		pomodoro_state INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNoteStoreAddNoteReturnsID(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	n := domnote.State{
		Note:      "test note",
		Pomodoro:  pomodoro.Work,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	id, err := ns.AddNote(context.Background(), n)
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}

	id2, err := ns.AddNote(context.Background(), n)
	if err != nil {
		t.Fatalf("AddNote 2: %v", err)
	}
	if id2 != 2 {
		t.Errorf("id = %d, want 2", id2)
	}
}

func TestNoteStoreGetNote(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	n := domnote.State{
		Note:      "hello",
		Pomodoro:  pomodoro.ShortBreak,
		CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	id, err := ns.AddNote(context.Background(), n)
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}

	got, err := ns.GetNote(context.Background(), id)
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if got.Note != "hello" {
		t.Errorf("Note = %q, want hello", got.Note)
	}
	if got.Pomodoro != pomodoro.ShortBreak {
		t.Errorf("Pomodoro = %d, want %d", got.Pomodoro, pomodoro.ShortBreak)
	}
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}
	wantTS := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if !got.CreatedAt.Equal(wantTS) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, wantTS)
	}
}

func TestNoteStoreGetNoteNotFound(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	_, err := ns.GetNote(context.Background(), 999)
	if !errors.Is(err, adaptNote.ErrNotFound) {
		t.Errorf("err = %v, want adaptNote.ErrNotFound", err)
	}
}

func TestNoteStoreDeleteNote(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	n := domnote.State{
		Note:     "to delete",
		Pomodoro: pomodoro.Work,
	}
	id, err := ns.AddNote(context.Background(), n)
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}

	if err := ns.DeleteNote(context.Background(), id); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	_, err = ns.GetNote(context.Background(), id)
	if !errors.Is(err, adaptNote.ErrNotFound) {
		t.Errorf("GetNote after delete: err = %v, want ErrNotFound", err)
	}
}

func TestNoteStoreDeleteNoteNotFound(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	err := ns.DeleteNote(context.Background(), 999)
	if !errors.Is(err, adaptNote.ErrNotFound) {
		t.Errorf("err = %v, want adaptNote.ErrNotFound", err)
	}
}

func TestNoteStoreListNotes(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	for i := 0; i < 5; i++ {
		n := domnote.State{
			Note:      "note",
			Pomodoro:  pomodoro.Work,
			CreatedAt: time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC),
		}
		_, err := ns.AddNote(context.Background(), n)
		if err != nil {
			t.Fatalf("AddNote %d: %v", i, err)
		}
	}

	list, err := ns.ListNotes(context.Background(), adaptNote.ListParams{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(list.Notes) != 3 {
		t.Errorf("len(Notes) = %d, want 3", len(list.Notes))
	}
	if list.Total != 5 {
		t.Errorf("Total = %d, want 5", list.Total)
	}
}

func TestNoteStoreListNotesByPomodoroState(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	work := pomodoro.Work
	breakState := pomodoro.ShortBreak

	for i := 0; i < 3; i++ {
		_, err := ns.AddNote(context.Background(), domnote.State{Note: "work", Pomodoro: work, CreatedAt: time.Now()})
		if err != nil {
			t.Fatalf("AddNote: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		_, err := ns.AddNote(context.Background(), domnote.State{Note: "break", Pomodoro: breakState, CreatedAt: time.Now()})
		if err != nil {
			t.Fatalf("AddNote: %v", err)
		}
	}

	list, err := ns.ListNotes(context.Background(), adaptNote.ListParams{PomodoroState: &breakState, Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(list.Notes) != 2 {
		t.Errorf("len(Notes) = %d, want 2", len(list.Notes))
	}
	if list.Total != 2 {
		t.Errorf("Total = %d, want 2", list.Total)
	}
}

func TestNoteStoreListNotesEmpty(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	list, err := ns.ListNotes(context.Background(), adaptNote.ListParams{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if list.Notes == nil {
		t.Error("Notes is nil, want empty slice")
	}
	if len(list.Notes) != 0 {
		t.Errorf("len(Notes) = %d, want 0", len(list.Notes))
	}
	if list.Total != 0 {
		t.Errorf("Total = %d, want 0", list.Total)
	}
}

func TestStorageEmitsDebugDiagnostics(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	n := domnote.State{
		Note:      "storage-log",
		Pomodoro:  pomodoro.Work,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	_, _ = ns.AddNote(context.Background(), n)
	_, _ = ns.GetNote(context.Background(), 1)
	_ = ns.DeleteNote(context.Background(), 1)
	_, _ = ns.ListNotes(context.Background(), adaptNote.ListParams{Limit: 10, Offset: 0})

	out := buf.String()
	for _, want := range []string{"sqlite: AddNote", "sqlite: GetNote", "sqlite: DeleteNote", "sqlite: ListNotes"} {
		if !strings.Contains(out, want) {
			t.Errorf("storage diagnostics missing %q, got:\n%s", want, out)
		}
	}
}
