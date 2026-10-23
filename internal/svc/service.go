package svc

import (
	"context"
	"errors"
	"fmt"
	"time"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
	oapi "github.com/4sp1/vipo-go/internal/port/oapi"
	xlate "github.com/4sp1/vipo-go/internal/port/oapi/http"
)

type Service struct {
	notes adaptNote.Store
	logs  adaptLog.EventLog
}

func New(notes adaptNote.Store, logs adaptLog.EventLog) *Service {
	return &Service{notes: notes, logs: logs}
}

// Compile-time check: Service satisfies StrictServerInterface.
var _ oapi.StrictServerInterface = (*Service)(nil)

func (s *Service) CreateNote(ctx context.Context, req oapi.CreateNoteRequestObject) (oapi.CreateNoteResponseObject, error) {
	if req.Body == nil {
		return nil, &ErrInvalidRequest{Message: "missing request body"}
	}

	ps, err := xlate.PomodoroStateToDomain(req.Body.PomodoroState)
	if err != nil {
		return nil, &ErrInvalidPomodoroState{Value: string(req.Body.PomodoroState)}
	}

	n := domnote.State{
		Note:      req.Body.Note,
		Pomodoro:  ps,
		CreatedAt: time.Now().UTC(),
	}

	id, err := s.notes.AddNote(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("add note: %w", err)
	}

	respPS, err := xlate.PomodoroStateFromDomain(ps)
	if err != nil {
		return nil, fmt.Errorf("convert pomodoro state: %w", err)
	}

	return oapi.CreateNote201JSONResponse{
		Id:            int64(id),
		Note:          n.Note,
		PomodoroState: respPS,
		CreatedAt:     n.CreatedAt,
	}, nil
}

func (s *Service) GetNote(ctx context.Context, req oapi.GetNoteRequestObject) (oapi.GetNoteResponseObject, error) {
	n, err := s.notes.GetNote(ctx, domnote.ID(req.Id))
	if err != nil {
		if errors.Is(err, adaptNote.ErrNotFound) {
			return nil, adaptNote.ErrNotFound
		}
		return nil, fmt.Errorf("get note: %w", err)
	}

	respPS, err := xlate.PomodoroStateFromDomain(n.Pomodoro)
	if err != nil {
		return nil, fmt.Errorf("convert pomodoro state: %w", err)
	}

	return oapi.GetNote200JSONResponse{
		Id:            int64(n.ID),
		Note:          n.Note,
		PomodoroState: respPS,
		CreatedAt:     n.CreatedAt,
	}, nil
}

func (s *Service) DeleteNote(ctx context.Context, req oapi.DeleteNoteRequestObject) (oapi.DeleteNoteResponseObject, error) {
	err := s.notes.DeleteNote(ctx, domnote.ID(req.Id))
	if err != nil {
		if errors.Is(err, adaptNote.ErrNotFound) {
			return nil, adaptNote.ErrNotFound
		}
		return nil, fmt.Errorf("delete note: %w", err)
	}

	return oapi.DeleteNote204Response{}, nil
}

func (s *Service) ListNotes(ctx context.Context, req oapi.ListNotesRequestObject) (oapi.ListNotesResponseObject, error) {
	limit := 50
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	offset := 0
	if req.Params.Offset != nil {
		offset = *req.Params.Offset
	}

	if limit < 1 || limit > 100 {
		return nil, &ErrInvalidLimit{Value: limit}
	}
	if offset < 0 {
		return nil, &ErrInvalidOffset{Value: offset}
	}

	var psFilter *pomodoro.State
	if req.Params.PomodoroState != nil {
		ps, err := xlate.PomodoroStateToDomain(*req.Params.PomodoroState)
		if err != nil {
			return nil, &ErrInvalidPomodoroState{Value: string(*req.Params.PomodoroState)}
		}
		psFilter = &ps
	}

	list, err := s.notes.ListNotes(ctx, adaptNote.ListParams{
		PomodoroState: psFilter,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}

	notes := make([]oapi.Note, 0, len(list.Notes))
	for _, n := range list.Notes {
		respPS, err := xlate.PomodoroStateFromDomain(n.Pomodoro)
		if err != nil {
			return nil, fmt.Errorf("convert pomodoro state: %w", err)
		}
		notes = append(notes, oapi.Note{
			Id:            int64(n.ID),
			Note:          n.Note,
			PomodoroState: respPS,
			CreatedAt:     n.CreatedAt,
		})
	}

	total := list.Total
	return oapi.ListNotes200JSONResponse{
		Notes: &notes,
		Total: &total,
	}, nil
}

func (s *Service) CreateLogEntry(ctx context.Context, req oapi.CreateLogEntryRequestObject) (oapi.CreateLogEntryResponseObject, error) {
	if req.Body == nil {
		return nil, &ErrInvalidRequest{Message: "missing request body"}
	}

	action, err := xlate.LogActionToDomain(req.Body.Action)
	if err != nil {
		return nil, &ErrInvalidLogAction{Value: string(req.Body.Action)}
	}

	session, err := xlate.SessionToDomain(req.Body.Session)
	if err != nil {
		return nil, &ErrInvalidSession{Value: string(req.Body.Session)}
	}

	var payload map[string]any
	if req.Body.Payload != nil {
		payload = *req.Body.Payload
	}

	entry := domlog.Entry{
		Action:    action,
		Session:   session,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}

	id, err := s.logs.AddEntry(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("add log entry: %w", err)
	}

	respAction, err := xlate.LogActionFromDomain(action)
	if err != nil {
		return nil, fmt.Errorf("convert action: %w", err)
	}
	respSession, err := xlate.SessionFromDomain(session)
	if err != nil {
		return nil, fmt.Errorf("convert session: %w", err)
	}

	var respPayload *map[string]any
	if payload != nil {
		respPayload = &payload
	}

	return oapi.CreateLogEntry201JSONResponse{
		Id:        int64(id),
		Action:    respAction,
		Session:   respSession,
		Payload:   respPayload,
		Timestamp: entry.Timestamp,
	}, nil
}

func (s *Service) ListLogEntries(ctx context.Context, req oapi.ListLogEntriesRequestObject) (oapi.ListLogEntriesResponseObject, error) {
	limit := 50
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	offset := 0
	if req.Params.Offset != nil {
		offset = *req.Params.Offset
	}

	if limit < 1 || limit > 100 {
		return nil, &ErrInvalidLimit{Value: limit}
	}
	if offset < 0 {
		return nil, &ErrInvalidOffset{Value: offset}
	}

	var actionFilter *domlog.Action
	if req.Params.Action != nil {
		action, err := xlate.LogActionToDomain(*req.Params.Action)
		if err != nil {
			return nil, &ErrInvalidLogAction{Value: string(*req.Params.Action)}
		}
		actionFilter = &action
	}

	var sessionFilter *domlog.Session
	if req.Params.Session != nil {
		session, err := xlate.SessionToDomain(*req.Params.Session)
		if err != nil {
			return nil, &ErrInvalidSession{Value: string(*req.Params.Session)}
		}
		sessionFilter = &session
	}

	var fromUTC, toUTC *time.Time
	if req.Params.From != nil {
		f := req.Params.From.UTC().Truncate(time.Second)
		fromUTC = &f
	}
	if req.Params.To != nil {
		to := req.Params.To.UTC()
		if to.Nanosecond() > 0 {
			to = to.Add(time.Second - time.Duration(to.Nanosecond())*time.Nanosecond)
		}
		toUTC = &to
	}

	list, err := s.logs.ListEntries(ctx, adaptLog.ListParams{
		Action:  actionFilter,
		Session: sessionFilter,
		From:    fromUTC,
		To:      toUTC,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list log entries: %w", err)
	}

	entries := make([]oapi.LogEntry, 0, len(list.Entries))
	for _, e := range list.Entries {
		respAction, err := xlate.LogActionFromDomain(e.Action)
		if err != nil {
			return nil, fmt.Errorf("convert action: %w", err)
		}
		respSession, err := xlate.SessionFromDomain(e.Session)
		if err != nil {
			return nil, fmt.Errorf("convert session: %w", err)
		}

		var respPayload *map[string]any
		if e.Payload != nil {
			respPayload = &e.Payload
		}

		entries = append(entries, oapi.LogEntry{
			Id:        int64(e.ID),
			Action:    respAction,
			Session:   respSession,
			Payload:   respPayload,
			Timestamp: e.Timestamp,
		})
	}

	total := list.Total
	return oapi.ListLogEntries200JSONResponse{
		Entries: &entries,
		Total:   &total,
	}, nil
}
