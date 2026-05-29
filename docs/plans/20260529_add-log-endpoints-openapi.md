# Add Log Endpoints OpenAPI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use supervised-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add POST endpoints to `openapi.json` for 6 log event actions (start, pause, reset, expire, resume, select), excluding `ActionNewNote` which is handled by notes CRUD.

**Architecture:** Each action gets its own POST endpoint under `/log/{action}`. Actions requiring session context (`start`, `select`) use request bodies with `PomodoroState`. Actions that deduct session from current state (`pause`, `reset`, `expire`, `resume`) use empty request bodies. All endpoints return `LogEntry` on 201.

**Tech Stack:** OpenAPI 3.0.3, JSON Schema

---

## Files

- **Modify:** `openapi.json` — Add schemas and paths for log endpoints

---

### Task 1: Add LogEntry Schema

**Files:**
- Modify: `openapi.json:183-263` (add new schemas in `components/schemas`)

- [ ] **Step 1: Add LogEntry response schema**

Add after the `Error` schema (line 262, before closing `}` of schemas):

```json
      "LogEntry": {
        "type": "object",
        "required": ["action", "session", "timestamp"],
        "properties": {
          "action": {
            "type": "string",
            "enum": ["start", "pause", "reset", "expire", "resume", "select"]
          },
          "session": {
            "$ref": "#/components/schemas/PomodoroState"
          },
          "timestamp": {
            "type": "string",
            "format": "date-time"
          }
        }
      }
```

- [ ] **Step 2: Verify JSON is valid**

Run: `python3 -c "import json; json.load(open('openapi.json'))"` or `jq . openapi.json > /dev/null`
Expected: No output (success) or valid JSON error if malformed

- [ ] **Step 3: Commit**

Message: `feat(openapi): add LogEntry schema for log events`

Use temp file pattern from core-commands skill:
```bash
mktemp
# Returns: /tmp/tmp.XXXXXX
```
Then use Write tool to save commit message to that path, then:
```bash
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

### Task 2: Add Request Schemas for Session-Payload Actions

**Files:**
- Modify: `openapi.json:183-263` (add new schemas after `LogEntry`)

- [ ] **Step 1: Add LogStartRequest schema**

Add after `LogEntry` schema:

```json
      "LogStartRequest": {
        "type": "object",
        "required": ["session"],
        "properties": {
          "session": {
            "$ref": "#/components/schemas/PomodoroState"
          }
        }
      }
```

- [ ] **Step 2: Add LogSelectRequest schema (identical structure)**

Add after `LogStartRequest` schema:

```json
      "LogSelectRequest": {
        "type": "object",
        "required": ["session"],
        "properties": {
          "session": {
            "$ref": "#/components/schemas/PomodoroState"
          }
        }
      }
```

- [ ] **Step 3: Verify JSON is valid**

Run: `jq . openapi.json > /dev/null && echo "Valid JSON"`
Expected: `Valid JSON`

- [ ] **Step 4: Commit**

Message: `feat(openapi): add LogStartRequest/LogSelectRequest schemas`

Use temp file pattern from core-commands skill:
```bash
mktemp
# Returns: /tmp/tmp.XXXXXX
```
Then use Write tool to save commit message to that path, then:
```bash
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

### Task 3: Add Request Schema for Actions Without Payload

**Files:**
- Modify: `openapi.json` (add new schema after `LogSelectRequest`)

- [ ] **Step 1: Add LogActionRequest schema (empty body)**

Add after `LogSelectRequest` schema:

```json
      "LogActionRequest": {
        "type": "object",
        "required": [],
        "properties": {}
      }
```

- [ ] **Step 2: Verify JSON is valid**

Run: `jq . openapi.json > /dev/null && echo "Valid JSON"`
Expected: `Valid JSON`

- [ ] **Step 3: Commit**

Message: `feat(openapi): add LogActionRequest schema for stateless actions`

Use temp file pattern from core-commands skill:
```bash
mktemp
# Returns: /tmp/tmp.XXXXXX
```
Then use Write tool to save commit message to that path, then:
```bash
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

### Task 4: Add POST /log/start Endpoint

**Files:**
- Modify: `openapi.json:14-181` (add new path in `paths` object)

- [ ] **Step 1: Add /log/start path definitionAfter the `/notes/{id}` path (after line 180, before closing `}` of paths):```json
    "/log/start": {
      "post": {
        "summary": "Log a start event",
        "operationId": "logStart",
        "tags": ["log"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/LogStartRequest"
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

Message: `feat(openapi): add POST /log/start endpoint`

Use temp file pattern from core-commands skill:
```bash
mktemp
# Returns: /tmp/tmp.XXXXXX
```
Then use Write tool to save commit message to that path, then:
```bash
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

### Task 5: Add POST /log/select Endpoint

**Files:**
- Modify: `openapi.json` (add new path after `/log/start`)

- [ ] **Step 1: Add /log/select path definition**

Add after `/log/start` path:

```json
    "/log/select": {
      "post": {
        "summary": "Log a select event",
        "operationId": "logSelect",
        "tags": ["log"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/LogSelectRequest"
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

Message: `feat(openapi): add POST /log/select endpoint`

Use temp file pattern from core-commands skill:
```bash
mktemp
# Returns: /tmp/tmp.XXXXXX
```
Then use Write tool to save commit message to that path, then:
```bash
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

### Task 6: Add POST /log/pause Endpoint

**Files:**
- Modify: `openapi.json` (add new path after `/log/select`)

- [ ] **Step 1: Add /log/pause path definition**

Add after `/log/select` path:

```json
    "/log/pause": {
      "post": {
        "summary": "Log a pause event",
        "operationId": "logPause",
        "tags": ["log"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/LogActionRequest"
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

Message: `feat(openapi): add POST /log/pause endpoint`

Use temp file pattern from core-commands skill:
```bash
mktemp
# Returns: /tmp/tmp.XXXXXX
```
Then use Write tool to save commit message to that path, then:
```bash
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

### Task 7: Add POST /log/reset Endpoint

**Files:**
- Modify: `openapi.json` (add new path after `/log/pause`)

- [ ] **Step 1: Add /log/reset path definition**

Add after `/log/pause` path:

```json
    "/log/reset": {
      "post": {
        "summary": "Log a reset event",
        "operationId": "logReset",
        "tags": ["log"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/LogActionRequest"
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

Message: `feat(openapi): add POST /log/reset endpoint`

Use temp file pattern from core-commands skill:
```bash
mktemp
# Returns: /tmp/tmp.XXXXXX
```
Then use Write tool to save commit message to that path, then:
```bash
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

### Task 8: Add POST /log/expire Endpoint

**Files:**
- Modify: `openapi.json` (add new path after `/log/reset`)

- [ ] **Step 1: Add /log/expire path definition**

Add after `/log/reset` path:

```json
    "/log/expire": {
      "post": {
        "summary": "Log an expire event",
        "operationId": "logExpire",
        "tags": ["log"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/LogActionRequest"
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

Message: `feat(openapi): add POST /log/expire endpoint`

Use temp file pattern from core-commands skill:
```bash
mktemp
# Returns: /tmp/tmp.XXXXXX
```
Then use Write tool to save commit message to that path, then:
```bash
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

### Task 9: Add POST /log/resume Endpoint

**Files:**
- Modify: `openapi.json` (add new path after `/log/expire`)

- [ ] **Step 1: Add /log/resume path definition**

Add after `/log/expire` path:

```json
    "/log/resume": {
      "post": {
        "summary": "Log a resume event",
        "operationId": "logResume",
        "tags": ["log"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/LogActionRequest"
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

Message: `feat(openapi): add POST /log/resume endpoint`

Use temp file pattern from core-commands skill:
```bash
mktemp
# Returns: /tmp/tmp.XXXXXX
```
Then use Write tool to save commit message to that path, then:
```bash
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

### Task 10: Final Verification

**Files:**
- Verify: `openapi.json`

- [ ] **Step 1: Validate complete OpenAPI schema**

Run: `jq . openapi.json > /dev/null && echo "Valid JSON"`
Expected: `Valid JSON`

- [ ] **Step 2: Verify all 6 endpoints present**

Run: `jq '.paths | keys' openapi.json`
Expected: Contains `/log/start`, `/log/select`, `/log/pause`, `/log/reset`, `/log/expire`, `/log/resume`

- [ ] **Step 3: Verify all schemas present**

Run: `jq '.components.schemas | keys' openapi.json`
Expected: Contains `LogEntry`, `LogStartRequest`, `LogSelectRequest`, `LogActionRequest`

- [ ] **Step 4: Verify LogEntry action enum excludes ActionNewNote**

Run: `jq '.components.schemas.LogEntry.properties.action.enum' openapi.json`
Expected: `["start", "pause", "reset", "expire", "resume", "select"]` (no `new_note`)

---

## Verification Checklist

- [ ] All 6 POST endpoints added (`/log/start`, `/log/select`, `/log/pause`, `/log/reset`, `/log/expire`, `/log/resume`)- [ ] `start` and `select` use `LogStartRequest`/`LogSelectRequest` with required `session`
- [ ] `pause`, `reset`, `expire`, `resume` use `LogActionRequest` (empty body)
- [ ] All return `LogEntry` on 201
- [ ] All use consistent error responses (400 Error)
- [ ] `LogEntry.action` enum matches domain `Action` constants (excluding `ActionNewNote`)
- [ ] JSON validates with `jq`