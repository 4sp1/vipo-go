package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
)

type eventLog struct {
	db *sql.DB
}

func (el *eventLog) AddEntry(ctx context.Context, entry domlog.Entry) (domlog.ID, error) {
	var payloadBytes []byte
	var err error
	if entry.Payload != nil {
		payloadBytes, err = json.Marshal(entry.Payload)
		if err != nil {
			return 0, fmt.Errorf("marshal payload: %w", err)
		}
	} else {
		payloadBytes = []byte("null")
	}

	result, err := el.db.ExecContext(ctx,
		"INSERT INTO log_entries (action, session, payload, timestamp) VALUES (?, ?, ?, ?)",
		int(entry.Action), int(entry.Session), string(payloadBytes), entry.Timestamp.Unix(),
	)
	if err != nil {
		return 0, fmt.Errorf("insert log entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return domlog.ID(id), nil
}

func (el *eventLog) ListEntries(ctx context.Context, params adaptLog.ListParams) (adaptLog.EntryList, error) {
	var conditions []string
	var args []any

	if params.Action != nil {
		conditions = append(conditions, "action = ?")
		args = append(args, int(*params.Action))
	}
	if params.Session != nil {
		conditions = append(conditions, "session = ?")
		args = append(args, int(*params.Session))
	}
	if params.From != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, params.From.Unix())
	}
	if params.To != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, params.To.Unix())
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM log_entries" + whereClause
	if err := el.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return adaptLog.EntryList{}, fmt.Errorf("count log entries: %w", err)
	}

	query := "SELECT id, action, session, payload, timestamp FROM log_entries" + whereClause + " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	rows, err := el.db.QueryContext(ctx, query, args...)
	if err != nil {
		return adaptLog.EntryList{}, fmt.Errorf("query log entries: %w", err)
	}
	defer rows.Close()

	entries := make([]domlog.Entry, 0)
	for rows.Next() {
		var e domlog.Entry
		var actionRaw, sessionRaw, timestampUnix int64
		var payloadStr string
		if err := rows.Scan(&e.ID, &actionRaw, &sessionRaw, &payloadStr, &timestampUnix); err != nil {
			return adaptLog.EntryList{}, fmt.Errorf("scan log entry: %w", err)
		}

		action, err := LogActionFromStorage(actionRaw)
		if err != nil {
			return adaptLog.EntryList{}, fmt.Errorf("convert action: %w", err)
		}
		session, err := LogSessionFromStorage(sessionRaw)
		if err != nil {
			return adaptLog.EntryList{}, fmt.Errorf("convert session: %w", err)
		}
		e.Action = action
		e.Session = session
		e.Timestamp = time.Unix(timestampUnix, 0)

		if payloadStr != "null" && payloadStr != "" {
			var payload map[string]any
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				return adaptLog.EntryList{}, fmt.Errorf("unmarshal payload: %w", err)
			}
			e.Payload = payload
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return adaptLog.EntryList{}, fmt.Errorf("rows: %w", err)
	}

	return adaptLog.EntryList{
		Entries: entries,
		Total:   total,
	}, nil
}
