# Plan: Add POST Operations for Log Events to OpenAPI

## Goal
Add POST endpoints to `openapi.json` for log events (excluding `ActionNewNote` which is handled by notes CRUD).

## Context

### Domain Structure (`internal/domain/state/log/entry.go`)
- **Actions**: `start`, `pause`, `reset`, `expire`, `resume`, `select` (excluding `new_note`)
- **Sessions**: `work`, `short_break`, `long_break` (map to `PomodoroState` enum)
- **Entry**: `Action` + `Session` + `Payload` + `Timestamp`

### User Requirements
- All 6 actions exposed via POST (including `expire`)
- **With session payload**: `start`, `select`
- **No session payload** (deducted from state): `pause`, `reset`, `expire`, `resume`
- **Endpoint style**: Separate paths per action (`/log/start`, `/log/pause`, etc.)

## Implementation

### 1. Add new schemas to `components/schemas`

#### `Session` enum (reuse existing `PomodoroState`)
Already exists in OpenAPI as `PomodoroState` with values `work`, `short_break`, `long_break`. Reference it.

#### `LogEntryStart` request schema
```json
{
  "type": "object",
  "required": ["session"],
  "properties": {
    "session": { "$ref": "#/components/schemas/PomodoroState" }
  }
}
```

#### `LogEntrySelect` request schema (same as start)
```json
{
  "type": "object",
  "required": ["session"],
  "properties": {
    "session": { "$ref": "#/components/schemas/PomodoroState" }
  }
}
```

#### `LogEntryAction` request schema (for pause/reset/expire/resume)
```json
{
  "type": "object",
  "required": [],
  "properties": {}
}
```
Empty object - no payload needed, session deducted from current state.

#### `LogEntry` response schema
```json
{
  "type": "object",
  "required": ["action", "session", "timestamp"],
  "properties": {
    "action": {
      "type": "string",
      "enum": ["start", "pause", "reset", "expire", "resume", "select"]
    },
    "session": { "$ref": "#/components/schemas/PomodoroState" },
    "timestamp": {
      "type": "string",
      "format": "date-time"
    }
  }
}
```

### 2. Add POST endpoints to `paths`

#### `/log/start`
- **Request body**: `LogEntryStart` (requires `session`)
- **Response**: 201 with `LogEntry`
- **OperationId**: `logStart`
- **Tag**: `log`

#### `/log/select`
- **Request body**: `LogEntrySelect` (requires `session`)
- **Response**: 201 with `LogEntry`
- **OperationId**: `logSelect`
- **Tag**: `log`

#### `/log/pause`
- **Request body**: `LogEntryAction` (empty)
- **Response**: 201 with `LogEntry`
- **OperationId**: `logPause`
- **Tag**: `log`

#### `/log/reset`
- **Request body**: `LogEntryAction` (empty)
- **Response**: 201 with `LogEntry`
- **OperationId**: `logReset`
- **Tag**: `log`

#### `/log/expire`
- **Request body**: `LogEntryAction` (empty)
- **Response**: 201 with `LogEntry`
- **OperationId**: `logExpire`
- **Tag**: `log`

#### `/log/resume`
- **Request body**: `LogEntryAction` (empty)
- **Response**: 201 with `LogEntry`
- **OperationId**: `logResume`
- **Tag**: `log`

## Verification
- [ ] All 6 POST endpoints added
- [ ] `start` and `select` require session in request body
- [ ] `pause`, `reset`, `expire`, `resume` have empty request body
- [ ] All return `LogEntry` on 201
- [ ] All use consistent error responses (400 Error)
- [ ] `ActionNewNote` is NOT included