package apperr

import (
	"errors"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	svcerrors "github.com/4sp1/vipo-go/internal/svc"
)

func Wire(err error) (code Code, message string, status int) {
	switch {
	case errors.Is(err, adaptNote.ErrNotFound):
		return CodeNoteNotFound, "note not found", 404
	case errors.Is(err, adaptLog.ErrNotFound):
		return CodeLogNotFound, "log entry not found", 404
	}

	var invalidPS *svcerrors.ErrInvalidPomodoroState
	if errors.As(err, &invalidPS) {
		return CodeInvalidPomodoroState, invalidPS.Error(), 400
	}

	var invalidAction *svcerrors.ErrInvalidLogAction
	if errors.As(err, &invalidAction) {
		return CodeInvalidLogAction, invalidAction.Error(), 400
	}

	var invalidSession *svcerrors.ErrInvalidSession
	if errors.As(err, &invalidSession) {
		return CodeInvalidPomodoroState, invalidSession.Error(), 400
	}

	var invalidLimit *svcerrors.ErrInvalidLimit
	if errors.As(err, &invalidLimit) {
		return CodeInvalidLimit, invalidLimit.Error(), 400
	}

	var invalidOffset *svcerrors.ErrInvalidOffset
	if errors.As(err, &invalidOffset) {
		return CodeInvalidOffset, invalidOffset.Error(), 400
	}

	var invalidReq *svcerrors.ErrInvalidRequest
	if errors.As(err, &invalidReq) {
		return CodeInvalidRequest, invalidReq.Error(), 400
	}

	return Code(-1), err.Error(), 500
}
