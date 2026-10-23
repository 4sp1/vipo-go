package note

import (
	"context"
	"testing"

	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

type mockStore struct {
	notes map[domnote.ID]domnote.State
}

func (m *mockStore) AddNote(_ context.Context, n domnote.State) (domnote.ID, error) {
	id := domnote.ID(len(m.notes) + 1)
	m.notes[id] = n
	return id, nil
}

func (m *mockStore) ListNotes(_ context.Context, _ ListParams) (List, error) {
	notes := make([]domnote.State, 0, len(m.notes))
	for _, n := range m.notes {
		notes = append(notes, n)
	}
	return List{Notes: notes, Total: len(notes)}, nil
}

func (m *mockStore) GetNote(_ context.Context, id domnote.ID) (domnote.State, error) {
	n, ok := m.notes[id]
	if !ok {
		return domnote.State{}, ErrNotFound
	}
	return n, nil
}

func (m *mockStore) DeleteNote(_ context.Context, id domnote.ID) error {
	delete(m.notes, id)
	return nil
}

func TestStoreInterface(t *testing.T) {
	var s Store = &mockStore{notes: make(map[domnote.ID]domnote.State)}
	n := domnote.State{Note: "test", Pomodoro: pomodoro.Work}
	id, err := s.AddNote(context.Background(), n)
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}
	got, err := s.GetNote(context.Background(), id)
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if got.Note != "test" {
		t.Errorf("Note = %q, want test", got.Note)
	}
}

func TestErrNotFound(t *testing.T) {
	s := &mockStore{notes: make(map[domnote.ID]domnote.State)}
	_, err := s.GetNote(context.Background(), 999)
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
