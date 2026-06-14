# Add Log Endpoints to OpenAPI Spec — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use supervised-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `POST /logs` and `GET /logs` endpoints to `openapi.json` with all required schemas for timer event tracking.

**Architecture:** Single `POST /logs` endpoint (action in request body) instead of per-action endpoints — all 6 actions create the same `LogEntry` resource, so one endpoint is sufficient. `GET /logs` with query params for filtering by action, session, and timestamp range. `LogSession` reuses `PomodoroState` schema since the values are identical. `LogAction` is a new enum with 6 values (excludes `unknown` and `new_note`).

**Tech Stack:** OpenAPI 3.0.3, JSON Schema

---

## Design Decisions

1. **`POST /logs` over per-action endpoints:** The issue poses "POST /logs or POST /logs/{action}". Six separate endpoints that all return `LogEntry` is boilerplate with no functional benefit. A single endpoint with `action` in the body matches the domain model (all entries have the same shape) and keeps the spec compact.

2. **`LogSession` reuses `PomodoroState`:** Both enumerate the same values (`work`, `short_break`, `long_break`). The `session` field on `LogEntry` references `PomodoroState` directly. The OpenAPI description documents the semantic distinction (timer state vs session context).

3. **`payload` as optional object:** The only payload type is `MessageNewNote`, which the notes API handles. Including `payload` as an optional `object` field on `CreateLogEntryRequest` and `LogEntry` keeps forward-compatibility without defining empty schemas.

4. **`ActionNewNote` excluded:** Per the success criteria, `ActionNewNote` is not in the `LogAction` enum. The `new_note` action is handled by `POST /notes`. `ActionUnknown` (iota 0) is also excluded as a zero-value sentinel.

5. **Timestamp format:** OpenAPI uses `date-time` (ISO8601). The adapter layer converts between ISO8601 strings and Unix integers at the DB boundary (per `AGENTS.md` convention).

---

## Files

- **Modify:** `openapi.json` — Add log schemas and path definitions

---

### Task 1: Add Log Schemas

**Files:**
- Modify: `openapi.json:249-263` (add new schemas after `Error`, before closing `}` of schemas)

- [ ] **Step 1: Add LogAction, CreateLogEntryRequest, LogEntry, and LogEntryList schemas**

Insert after the `Error` schema (after line 262, before the closing `}` of the `schemas` object):

```json
      "LogAction": {
        "type": "string",
        "enum": ["start", "pause", "reset", "expire", "resume", "select"],
        "description": "Timer event action. 'new_note' is excluded — use POST /notes instead."
      },
      "CreateLogEntryRequest": {
        "type": "object",
        "required": ["action", "session"],
        "properties": {
          "action": {
            "$ref": "#/components/schemas/LogAction"
          },
          "session": {
            "$ref": "#/components/schemas/PomodoroState"
          },
          "payload": {
            "type": "object",
            "description": "Optional action-specific data"
          }
        }
      },
      "LogEntry": {
        "type": "object",
        "required": ["id", "action", "session", "timestamp"],
        "properties": {
          "id": {
            "type": "integer",
            "format": "int64"
          },
          "action": {
            "$ref": "#/components/schemas/LogAction"
          },
          "session": {
            "$ref": "#/components/schemas/PomodoroState"
          },
          "payload": {
            "type": "object",
            "description": "Optional action-specific data"
          },
          "timestamp": {
            "type": "string",
            "format": "date-time"
          }
        }
      },
      "LogEntryList": {
        "type": "object",
        "properties": {
          "entries": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/LogEntry"
            }
          },
          "total": {
            "type": "integer"
          }
        }
      }
```

- [ ] **Step 2: Verify JSON is valid**

Run: `jq . openapi.json > /dev/null && echo "Valid JSON"`
Expected: `Valid JSON`

- [ ] **Step 3: Commit**

```
feat(openapi): add LogAction, LogEntry, and LogEntryList schemas

Refs #8
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

### Task 2: Add POST /logs Endpoint

**Files:**
- Modify: `openapi.json:14-181` (add new path in `paths` object)

- [ ] **Step 1: Add /logs path with POST operation**

Insert after the `/notes/{id}` path (after line 180, before closing `}` of paths), adding a comma after the `/notes/{id}` closing brace:

```json
    "/logs": {
      "post": {
        "summary": "Create a log entry",
        "operationId": "createLogEntry",
        "tags": [
          "logs"
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/CreateLogEntryRequest"
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Log entry created",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/LogEntry"
                }
              }
            }
          },
          "400": {
            "description": "Invalid request",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Error"
                }
              }
            }
          }
        }
      }
    }
```

- [ ] **Step 2: Verify JSON is valid**

Run: `jq . openapi.json > /dev/null && echo "Valid JSON"`
Expected: `Valid JSON`

- [ ] **Step 3: Commit**

```
feat(openapi): add POST /logs endpoint

Refs #8
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

### Task 3: Add GET /logs Endpoint

**Files:**
- Modify: `openapi.json` (add GET operation to the `/logs` path added in Task 2)

- [ ] **Step 1: Add GET operation to /logs path**

Add the `get` key alongside the existing `post` key inside the `/logs` path object:

```json
      "get": {
        "summary": "List log entries",
        "operationId": "listLogEntries",
        "tags": [
          "logs"
        ],
        "parameters": [
          {
            "name": "action",
            "in": "query",
            "required": false,
            "schema": {
              "$ref": "#/components/schemas/LogAction"
            }
          },
          {
            "name": "session",
            "in": "query",
            "required": false,
            "schema": {
              "$ref": "#/components/schemas/PomodoroState"
            }
          },
          {
            "name": "from",
            "in": "query",
            "required": false,
            "schema": {
              "type": "string",
              "format": "date-time"
            },
            "description": "Filter entries from this timestamp (inclusive)"
          },
          {
            "name": "to",
            "in": "query",
            "required": false,
            "schema": {
              "type": "string",
              "format": "date-time"
            },
            "description": "Filter entries up to this timestamp (inclusive)"
          },
          {
            "name": "limit",
            "in": "query",
            "required": false,
            "schema": {
              "type": "integer",
              "minimum": 1,
              "maximum": 100,
              "default": 50
            }
          },
          {
            "name": "offset",
            "in": "query",
            "required": false,
            "schema": {
              "type": "integer",
              "minimum": 0,
              "default": 0
            }
          }
        ],
        "responses": {
          "200": {
            "description": "List of log entries",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/LogEntryList"
                }
              }
            }
          }
        }
      }
```

- [ ] **Step 2: Verify JSON is valid**

Run: `jq . openapi.json > /dev/null && echo "Valid JSON"`
Expected: `Valid JSON`

- [ ] **Step 3: Commit**

```
feat(openapi): add GET /logs endpoint with timestamp filtering

Refs #8
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

## Success Criteria Verification

| Criterion | Covered By |
|-----------|-----------|
| `LogAction` schema with 6 non-unknown actions | Task 1 (`start`, `pause`, `reset`, `expire`, `resume`, `select`) |
| `LogSession` reuses `PomodoroState` | Task 1 (documented in `LogEntry.session` via `$ref`) |
| `LogEntry` schema matches domain fields | Task 1 (`id`, `action`, `session`, `payload`, `timestamp`) |
| `POST /logs` endpoint defined | Task 2 |
| `GET /logs` with `from`/`to` query params | Task 3 (also `action`, `session`, `limit`, `offset`) |
| `ActionNewNote` excluded from HTTP surface | Task 1 (not in `LogAction` enum, noted in description) |