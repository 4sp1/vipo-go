package apperr

import (
	"errors"
	"testing"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	svcerrors "github.com/4sp1/vipo-go/internal/svc"
)

func TestWireNoteNotFound(t *testing.T) {
	code, msg, status := Wire(adaptNote.ErrNotFound)
	if code != CodeNoteNotFound {
		t.Errorf("code = %s, want %s", code, CodeNoteNotFound)
	}
	if status != 404 {
		t.Errorf("status = %d, want 404", status)
	}
	if msg == "" {
		t.Error("message should not be empty")
	}
}

func TestWireLogNotFound(t *testing.T) {
	code, msg, status := Wire(adaptLog.ErrNotFound)
	if code != CodeLogNotFound {
		t.Errorf("code = %s, want %s", code, CodeLogNotFound)
	}
	if status != 404 {
		t.Errorf("status = %d, want 404", status)
	}
	if msg == "" {
		t.Error("message should not be empty")
	}
}

func TestWireInvalidPomodoroState(t *testing.T) {
	err := &svcerrors.ErrInvalidPomodoroState{Value: "bad"}
	code, _, status := Wire(err)
	if code != CodeInvalidPomodoroState {
		t.Errorf("code = %s, want %s", code, CodeInvalidPomodoroState)
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestWireInvalidLogAction(t *testing.T) {
	err := &svcerrors.ErrInvalidLogAction{Value: "bad"}
	code, _, status := Wire(err)
	if code != CodeInvalidLogAction {
		t.Errorf("code = %s, want %s", code, CodeInvalidLogAction)
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestWireInvalidSession(t *testing.T) {
	err := &svcerrors.ErrInvalidSession{Value: "bad"}
	code, _, status := Wire(err)
	if code != CodeInvalidPomodoroState {
		t.Errorf("code = %s, want %s", code, CodeInvalidPomodoroState)
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestWireInvalidLimit(t *testing.T) {
	err := &svcerrors.ErrInvalidLimit{Value: 0}
	code, _, status := Wire(err)
	if code != CodeInvalidLimit {
		t.Errorf("code = %s, want %s", code, CodeInvalidLimit)
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestWireInvalidOffset(t *testing.T) {
	err := &svcerrors.ErrInvalidOffset{Value: -1}
	code, _, status := Wire(err)
	if code != CodeInvalidOffset {
		t.Errorf("code = %s, want %s", code, CodeInvalidOffset)
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestWireInvalidRequest(t *testing.T) {
	err := &svcerrors.ErrInvalidRequest{Message: "missing body"}
	code, _, status := Wire(err)
	if code != CodeInvalidRequest {
		t.Errorf("code = %s, want %s", code, CodeInvalidRequest)
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestWireUnknownError(t *testing.T) {
	err := errors.New("something broke")
	code, _, status := Wire(err)
	if code != Code(-1) {
		t.Errorf("code = %s, want Code(-1)", code)
	}
	if status != 500 {
		t.Errorf("status = %d, want 500", status)
	}
}
