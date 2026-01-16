# History Immediate Logging

Improve history logging to write entries immediately when commands start and update them on completion.

## Problem

Current implementation only writes history AFTER command completion:
- Crashes/interrupts lose the record entirely
- No visibility into running commands
- No unique identifier to track entries

## Solution

### New Entry Structure

```yaml
entries:
  - id: swift_falcon_20251216_180816
    command: plan
    spec: 035-docs-man-command
    status: completed  # running | completed | failed | cancelled
    created_at: 2025-12-16T18:08:16-08:00
    completed_at: 2025-12-16T18:09:51-08:00
    exit_code: 0
    duration: 1m35.923s
```

### ID Format

`adjective_noun_YYYYMMDD_HHMMSS` - memorable word pairs + timestamp

Examples: `brave_fox_20251216_180816`, `calm_river_20251216_183042`

## Tasks

- [ ] Add word lists for ID generation (`internal/history/words.go`)
  - ~50 adjectives (brave, calm, swift, bright, etc.)
  - ~50 nouns (fox, river, falcon, oak, etc.)
  - `GenerateID()` function

- [ ] Update `HistoryEntry` struct (`internal/history/history.go`)
  - Add `ID string`
  - Add `Status string` (running, completed, failed, cancelled)
  - Rename `Timestamp` to `CreatedAt`
  - Add `CompletedAt time.Time`
  - Keep `Duration` as computed on completion

- [ ] Update `Writer` API (`internal/history/writer.go`)
  - `StartCommand(command, spec string) string` - writes entry with status=running, returns ID
  - `CompleteCommand(id string, exitCode int)` - updates entry with status, exit_code, completed_at, duration
  - `FindByID(id string) *HistoryEntry` - helper to find entry

- [ ] Update `HistoryLogger` interface (`internal/lifecycle/handler.go`)
  - Change from single `LogCommand()` to `StartCommand()` + `CompleteCommand()`

- [ ] Update lifecycle wrappers (`internal/lifecycle/lifecycle.go`)
  - Call `StartCommand()` at beginning, get ID
  - Call `CompleteCommand(id)` at end

- [ ] Update history display (`internal/cli/history.go`)
  - Show status column
  - Color code: green=completed, yellow=running, red=failed
  - Support filtering by status `--status running`

- [ ] Update docs (`docs/reference.md`)
  - Document new history format
  - Document new --status flag

- [ ] Tests
  - Test ID generation uniqueness
  - Test start/complete flow
  - Test crash recovery (entries left in "running" state)
