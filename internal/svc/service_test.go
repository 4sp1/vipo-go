package svc

import (
	"context"
	"errors"
	"testing"
	"time"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
	oapi "github.com/4sp1/vipo-go/internal/port/oapi"
)

type recordingNoteStore struct {
	notes  map[domnote.ID]domnote.State
	nextID domnote.ID
}

func newRecordingNoteStore() *recordingNoteStore {
	return &recordingNoteStore{notes: make(map[domnote.ID]domnote.State), nextID: 1}
}

func (s *recordingNoteStore) AddNote(_ context.Context, n domnote.State) (domnote.ID, error) {
	id := s.nextID
	s.nextID++
	n.ID = id
	s.notes[id] = n
	return id, nil
}

func (s *recordingNoteStore) ListNotes(_ context.Context, params adaptNote.ListParams) (adaptNote.List, error) {
	var notes []domnote.State
	for _, n := range s.notes {
		if params.PomodoroState != nil && n.Pomodoro != *params.PomodoroState {
			continue
		}
		notes = append(notes, n)
	}
	total := len(notes)
	if params.Offset >= len(notes) {
		notes = nil
	} else {
		notes = notes[params.Offset:]
	}
	if params.Limit < len(notes) {
		notes = notes[:params.Limit]
	}
	if notes == nil {
		notes = []domnote.State{}
	}
	return adaptNote.List{Notes: notes, Total: total}, nil
}

func (s *recordingNoteStore) GetNote(_ context.Context, id domnote.ID) (domnote.State, error) {
	n, ok := s.notes[id]
	if !ok {
		return domnote.State{}, adaptNote.ErrNotFound
	}
	return n, nil
}

func (s *recordingNoteStore) DeleteNote(_ context.Context, id domnote.ID) error {
	if _, ok := s.notes[id]; !ok {
		return adaptNote.ErrNotFound
	}
	delete(s.notes, id)
	return nil
}

type stubEventLog struct {
	adaptLog.EventLog
}

func TestNewService(t *testing.T) {
	s := New(&recordingNoteStore{}, &stubEventLog{})
	if s == nil {
		t.Fatal("New returned nil")
	}
}

func TestCreateNote(t *testing.T) {
	tests := []struct {
		name     string
		body     oapi.CreateNoteRequest
		wantErr  bool
		wantNote string
	}{
		{
			name:     "valid work note",
			body:     oapi.CreateNoteRequest{Note: "hello", PomodoroState: oapi.Work},
			wantNote: "hello",
		},
		{
			name:     "valid short_break note",
			body:     oapi.CreateNoteRequest{Note: "break time", PomodoroState: oapi.ShortBreak},
			wantNote: "break time",
		},
		{
			name:    "invalid pomodoro state",
			body:    oapi.CreateNoteRequest{Note: "test", PomodoroState: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newRecordingNoteStore()
			s := New(store, &stubEventLog{})

			resp, err := s.CreateNote(context.Background(), oapi.CreateNoteRequestObject{
				Body: &tt.body,
			})

			if tt.wantErr {
				if _, ok := resp.(oapi.CreateNote400JSONResponse); !ok {
					t.Fatalf("expected 400 response, got %T", resp)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateNote: %v", err)
			}

			createResp, ok := resp.(oapi.CreateNote201JSONResponse)
			if !ok {
				t.Fatalf("unexpected response type: %T", resp)
			}
			if createResp.Note != tt.wantNote {
				t.Errorf("Note = %q, want %q", createResp.Note, tt.wantNote)
			}
			if createResp.PomodoroState != tt.body.PomodoroState {
				t.Errorf("PomodoroState = %q, want %q", createResp.PomodoroState, tt.body.PomodoroState)
			}
			if createResp.Id != 1 {
				t.Errorf("Id = %d, want 1", createResp.Id)
			}
		})
	}
}

func TestGetNote(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	n := domnote.State{
		Note:      "fetch me",
		Pomodoro:  pomodoro.LongBreak,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	id, _ := store.AddNote(context.Background(), n)

	resp, err := s.GetNote(context.Background(), oapi.GetNoteRequestObject{Id: int64(id)})
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}

	getResp, ok := resp.(oapi.GetNote200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if getResp.Note != "fetch me" {
		t.Errorf("Note = %q, want 'fetch me'", getResp.Note)
	}
	if getResp.PomodoroState != oapi.LongBreak {
		t.Errorf("PomodoroState = %q, want %q", getResp.PomodoroState, oapi.LongBreak)
	}
}

func TestGetNoteNotFound(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	resp, err := s.GetNote(context.Background(), oapi.GetNoteRequestObject{Id: 999})
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}

	_, ok := resp.(oapi.GetNote404JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T, want GetNote404JSONResponse", resp)
	}
}

func TestDeleteNote(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	n := domnote.State{
		Note:     "delete me",
		Pomodoro: pomodoro.Work,
	}
	id, _ := store.AddNote(context.Background(), n)

	resp, err := s.DeleteNote(context.Background(), oapi.DeleteNoteRequestObject{Id: int64(id)})
	if err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	_, ok := resp.(oapi.DeleteNote204Response)
	if !ok {
		t.Fatalf("unexpected response type: %T, want DeleteNote204Response", resp)
	}

	_, err = store.GetNote(context.Background(), id)
	if !errors.Is(err, adaptNote.ErrNotFound) {
		t.Errorf("GetNote after delete: %v, want ErrNotFound", err)
	}
}

func TestDeleteNoteNotFound(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	resp, err := s.DeleteNote(context.Background(), oapi.DeleteNoteRequestObject{Id: 999})
	if err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	_, ok := resp.(oapi.DeleteNote404JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T, want DeleteNote404JSONResponse", resp)
	}
}

func TestListNotesDefaultLimitOffset(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	for i := 0; i < 60; i++ {
		store.AddNote(context.Background(), domnote.State{
			Note:     "note",
			Pomodoro: pomodoro.Work,
		})
	}

	resp, err := s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}

	listResp, ok := resp.(oapi.ListNotes200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if listResp.Notes == nil {
		t.Fatal("Notes is nil")
	}
	if len(*listResp.Notes) != 50 {
		t.Errorf("len(Notes) = %d, want 50", len(*listResp.Notes))
	}
	if listResp.Total == nil {
		t.Fatal("Total is nil")
	}
	if *listResp.Total != 60 {
		t.Errorf("Total = %d, want 60", *listResp.Total)
	}
}

func TestListNotesInvalidLimit(t *testing.T) {
	s := New(newRecordingNoteStore(), &stubEventLog{})

	badLimit := 0
	resp, err := s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{Limit: &badLimit},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	_, ok := resp.(listNotes400Response)
	if !ok {
		t.Fatalf("expected 400 for limit=0, got %T", resp)
	}

	bigLimit := 101
	resp, err = s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{Limit: &bigLimit},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	_, ok = resp.(listNotes400Response)
	if !ok {
		t.Fatalf("expected 400 for limit=101, got %T", resp)
	}
}

func TestListNotesInvalidOffset(t *testing.T) {
	s := New(newRecordingNoteStore(), &stubEventLog{})

	badOffset := -1
	resp, err := s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{Offset: &badOffset},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	_, ok := resp.(listNotes400Response)
	if !ok {
		t.Fatalf("expected 400 for offset=-1, got %T", resp)
	}
}

func TestListNotesByPomodoroState(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	store.AddNote(context.Background(), domnote.State{Note: "work", Pomodoro: pomodoro.Work})
	store.AddNote(context.Background(), domnote.State{Note: "break", Pomodoro: pomodoro.ShortBreak})
	store.AddNote(context.Background(), domnote.State{Note: "work2", Pomodoro: pomodoro.Work})

	resp, err := s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{PomodoroState: &[]oapi.PomodoroState{oapi.Work}[0]},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}

	listResp, ok := resp.(oapi.ListNotes200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Notes) != 2 {
		t.Errorf("len(Notes) = %d, want 2", len(*listResp.Notes))
	}
	if *listResp.Total != 2 {
		t.Errorf("Total = %d, want 2", *listResp.Total)
	}
}

func TestListNotesInvalidPomodoroState(t *testing.T) {
	s := New(newRecordingNoteStore(), &stubEventLog{})

	badState := oapi.PomodoroState("invalid")
	resp, err := s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{PomodoroState: &badState},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	_, ok := resp.(listNotes400Response)
	if !ok {
		t.Fatalf("expected 400 for invalid pomodoro_state, got %T", resp)
	}
}

type recordingEventLog struct {
	entries map[domlog.ID]domlog.Entry
	nextID  domlog.ID
}

func newRecordingEventLog() *recordingEventLog {
	return &recordingEventLog{entries: make(map[domlog.ID]domlog.Entry), nextID: 1}
}

func (e *recordingEventLog) AddEntry(_ context.Context, entry domlog.Entry) (domlog.ID, error) {
	id := e.nextID
	e.nextID++
	entry.ID = id
	e.entries[id] = entry
	return id, nil
}

func (e *recordingEventLog) ListEntries(_ context.Context, params adaptLog.ListParams) (adaptLog.EntryList, error) {
	var entries []domlog.Entry
	for _, e := range e.entries {
		if params.Action != nil && e.Action != *params.Action {
			continue
		}
		if params.Session != nil && e.Session != *params.Session {
			continue
		}
		if params.From != nil && e.Timestamp.Before(*params.From) {
			continue
		}
		if params.To != nil && e.Timestamp.After(*params.To) {
			continue
		}
		entries = append(entries, e)
	}
	total := len(entries)
	if params.Offset >= len(entries) {
		entries = nil
	} else {
		entries = entries[params.Offset:]
	}
	if params.Limit < len(entries) {
		entries = entries[:params.Limit]
	}
	if entries == nil {
		entries = []domlog.Entry{}
	}
	return adaptLog.EntryList{Entries: entries, Total: total}, nil
}

func TestCreateLogEntry(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	payload := map[string]any{"key": "value"}
	body := oapi.CreateLogEntryRequest{
		Action:  oapi.Start,
		Session: oapi.Work,
		Payload: &payload,
	}

	resp, err := s.CreateLogEntry(context.Background(), oapi.CreateLogEntryRequestObject{
		Body: &body,
	})
	if err != nil {
		t.Fatalf("CreateLogEntry: %v", err)
	}

	createResp, ok := resp.(oapi.CreateLogEntry201JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if createResp.Id != 1 {
		t.Errorf("Id = %d, want 1", createResp.Id)
	}
	if createResp.Action != oapi.Start {
		t.Errorf("Action = %q, want %q", createResp.Action, oapi.Start)
	}
	if createResp.Session != oapi.Work {
		t.Errorf("Session = %q, want %q", createResp.Session, oapi.Work)
	}
}

func TestCreateLogEntryInvalidAction(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog())

	body := oapi.CreateLogEntryRequest{
		Action:  "invalid",
		Session: oapi.Work,
	}

	resp, err := s.CreateLogEntry(context.Background(), oapi.CreateLogEntryRequestObject{
		Body: &body,
	})
	if err != nil {
		t.Fatalf("CreateLogEntry: %v", err)
	}
	_, ok := resp.(oapi.CreateLogEntry400JSONResponse)
	if !ok {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestCreateLogEntryInvalidSession(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog())

	body := oapi.CreateLogEntryRequest{
		Action:  oapi.Start,
		Session: "invalid",
	}

	resp, err := s.CreateLogEntry(context.Background(), oapi.CreateLogEntryRequestObject{
		Body: &body,
	})
	if err != nil {
		t.Fatalf("CreateLogEntry: %v", err)
	}
	_, ok := resp.(oapi.CreateLogEntry400JSONResponse)
	if !ok {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestCreateLogEntryNilPayload(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	body := oapi.CreateLogEntryRequest{
		Action:  oapi.Start,
		Session: oapi.Work,
	}

	resp, err := s.CreateLogEntry(context.Background(), oapi.CreateLogEntryRequestObject{
		Body: &body,
	})
	if err != nil {
		t.Fatalf("CreateLogEntry: %v", err)
	}

	createResp, ok := resp.(oapi.CreateLogEntry201JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if createResp.Payload != nil {
		t.Errorf("Payload = %v, want nil", createResp.Payload)
	}
}

func TestListLogEntries(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	for i := 0; i < 5; i++ {
		logs.AddEntry(context.Background(), domlog.Entry{
			Action:    domlog.ActionStart,
			Session:   domlog.SessionWork,
			Payload:   map[string]any{"index": i},
			Timestamp: time.Date(2026, 1, 1, 0, i, 0, 0, time.UTC),
		})
	}

	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if listResp.Entries == nil {
		t.Fatal("Entries is nil")
	}
	if len(*listResp.Entries) != 5 {
		t.Errorf("len(Entries) = %d, want 5", len(*listResp.Entries))
	}
	if listResp.Total == nil || *listResp.Total != 5 {
		t.Errorf("Total = %v, want 5", listResp.Total)
	}
}

func TestListLogEntriesDefaultLimitOffset(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	for i := 0; i < 60; i++ {
		logs.AddEntry(context.Background(), domlog.Entry{
			Action:    domlog.ActionStart,
			Session:   domlog.SessionWork,
			Timestamp: time.Now(),
		})
	}

	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Entries) != 50 {
		t.Errorf("len(Entries) = %d, want 50 (default)", len(*listResp.Entries))
	}
	if *listResp.Total != 60 {
		t.Errorf("Total = %d, want 60", *listResp.Total)
	}
}

func TestListLogEntriesInvalidLimit(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog())

	badLimit := 0
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Limit: &badLimit},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}
	_, ok := resp.(listLogEntries400Response)
	if !ok {
		t.Fatalf("expected 400 for limit=0, got %T", resp)
	}
}

func TestListLogEntriesInvalidOffset(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog())

	badOffset := -1
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Offset: &badOffset},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}
	_, ok := resp.(listLogEntries400Response)
	if !ok {
		t.Fatalf("expected 400 for offset=-1, got %T", resp)
	}
}

func TestListLogEntriesFilterByAction(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionWork, Timestamp: time.Now()})
	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionPause, Session: domlog.SessionWork, Timestamp: time.Now()})

	action := oapi.Start
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Action: &action},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(*listResp.Entries))
	}
}

func TestListLogEntriesFilterBySession(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionWork, Timestamp: time.Now()})
	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionShortBreak, Timestamp: time.Now()})

	session := oapi.Work
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Session: &session},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(*listResp.Entries))
	}
}

func TestListLogEntriesFilterByTimeRange(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionWork, Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionWork, Timestamp: time.Date(2026, 1, 1, 0, 1, 30, 0, time.UTC)})
	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionWork, Timestamp: time.Date(2026, 1, 1, 0, 3, 0, 0, time.UTC)})

	from := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC)

	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{From: &from, To: &to},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(*listResp.Entries))
	}
}

func TestListLogEntriesInvalidAction(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog())

	badAction := oapi.LogAction("invalid")
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Action: &badAction},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}
	_, ok := resp.(listLogEntries400Response)
	if !ok {
		t.Fatalf("expected 400 for invalid action, got %T", resp)
	}
}

func TestListLogEntriesInvalidSession(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog())

	badSession := oapi.PomodoroState("invalid")
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Session: &badSession},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}
	_, ok := resp.(listLogEntries400Response)
	if !ok {
		t.Fatalf("expected 400 for invalid session, got %T", resp)
	}
}
