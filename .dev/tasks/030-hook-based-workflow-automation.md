# Hook-Based Workflow Automation with autospec Notifications

This document explores how to leverage autospec's notification hook system (`OnCommandComplete`, `OnStageComplete`, `OnError`) to trigger automated dev workflow actions.

---

## Current Hook System Overview

autospec provides three hook functions in `internal/notify/handler.go`:

| Hook | Trigger | Config Key |
|------|---------|------------|
| `OnCommandComplete` | Any autospec command finishes | `on_command_complete` |
| `OnStageComplete` | A workflow stage completes | `on_stage_complete` |
| `OnError` | A command or stage fails | `on_error` |

### Configuration

```yaml
# ~/.config/autospec/config.yml or .autospec/config.yml
notifications:
  enabled: true
  type: both                    # sound, visual, or both
  sound_file: "/path/to/custom.wav"
  on_command_complete: true     # fires on any command finish
  on_stage_complete: false      # fires after specify, plan, tasks, implement stages
  on_error: true                # fires on failures
  on_long_running: false        # only notify if duration > threshold
  long_running_threshold: 30s
```

---

## Proposed Hook Extensions for Workflow Automation

The current hooks send notifications (sound/visual). We can extend this system to execute arbitrary shell commands, enabling powerful workflow automation.

### Proposed Config Schema

```yaml
notifications:
  enabled: true
  hooks:
    on_command_complete:
      - command: "make lint"
        only_on_success: true
      - command: "notify-send 'autospec done'"

    on_stage_complete:
      - command: "make test"
        stages: ["implement"]     # only after implement stage
      - command: "git status"
        stages: ["*"]             # all stages

    on_error:
      - command: "make fmt"       # auto-fix formatting on failure
      - command: "echo 'Failed: $AUTOSPEC_COMMAND' >> ~/.autospec/errors.log"
```

### Environment Variables for Hooks

Hooks would receive context via environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `AUTOSPEC_COMMAND` | Command that triggered hook | `implement` |
| `AUTOSPEC_STAGE` | Current workflow stage | `tasks` |
| `AUTOSPEC_SPEC` | Spec name | `003-auth` |
| `AUTOSPEC_SUCCESS` | Exit status | `true` / `false` |
| `AUTOSPEC_DURATION` | Execution time | `45.2s` |
| `AUTOSPEC_EXIT_CODE` | Numeric exit code | `0`, `1`, `2` |

---

## Common Workflow Hook Ideas

### 1. Auto-Linting After Implementation

Automatically run linters when implementation completes:

```yaml
hooks:
  on_stage_complete:
    - command: "make lint"
      stages: ["implement"]
      only_on_success: true
```

Or fix common issues automatically:

```yaml
hooks:
  on_error:
    - command: "make fmt && goimports -w ."
      description: "Auto-fix formatting issues"
```

### 2. Build Verification

Trigger builds after code changes:

```yaml
hooks:
  on_command_complete:
    - command: "make build"
      commands: ["implement", "run"]
      only_on_success: true
    - command: "go vet ./..."
      commands: ["implement"]
```

### 3. Test Running

Run tests at different granularities:

```yaml
hooks:
  on_stage_complete:
    # Fast tests after each stage
    - command: "go test -short ./..."
      stages: ["*"]

    # Full test suite only after implement
    - command: "make test"
      stages: ["implement"]
      only_on_success: true
```

### 4. Git Operations

Auto-stage or commit after successful implementation:

```yaml
hooks:
  on_stage_complete:
    - command: "git add -A && git status"
      stages: ["implement"]
      only_on_success: true

    # Optional: auto-commit (use with caution!)
    - command: "git commit -m 'feat($AUTOSPEC_SPEC): implement feature'"
      stages: ["implement"]
      only_on_success: true
      require_confirmation: true  # prompt before executing
```

### 5. Documentation Generation

Regenerate docs after spec changes:

```yaml
hooks:
  on_stage_complete:
    - command: "make docs"
      stages: ["specify", "plan"]
    - command: "godoc -http=:6060 &"
      stages: ["implement"]
```

### 6. CI/CD Integration

Trigger CI pipelines or webhooks:

```yaml
hooks:
  on_command_complete:
    - command: "curl -X POST https://ci.example.com/trigger -d '{\"spec\": \"$AUTOSPEC_SPEC\"}'"
      commands: ["implement"]
      only_on_success: true

    - command: "gh workflow run tests.yml"
      commands: ["implement"]
```

### 7. Notification Integrations

Send to various notification systems:

```yaml
hooks:
  on_command_complete:
    # Slack webhook
    - command: |
        curl -X POST $SLACK_WEBHOOK -d '{
          "text": "autospec $AUTOSPEC_COMMAND completed ($AUTOSPEC_DURATION)"
        }'

    # Desktop notification (Linux)
    - command: "notify-send 'autospec' '$AUTOSPEC_COMMAND finished'"

    # macOS notification
    - command: "osascript -e 'display notification \"$AUTOSPEC_COMMAND done\" with title \"autospec\"'"

    # ntfy.sh (self-hosted push notifications)
    - command: "curl -d '$AUTOSPEC_COMMAND completed' ntfy.sh/my-autospec-topic"
```

### 8. Quality Gates

Enforce quality standards before proceeding:

```yaml
hooks:
  on_stage_complete:
    - command: |
        # Block if coverage drops below threshold
        go test -coverprofile=coverage.out ./...
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
        if (( $(echo "$COVERAGE < 80" | bc -l) )); then
          echo "Coverage $COVERAGE% below 80% threshold"
          exit 1
        fi
      stages: ["implement"]
      block_on_failure: true  # prevent next stage if this fails
```

### 9. File Watching / Hot Reload

Restart services or watchers:

```yaml
hooks:
  on_stage_complete:
    - command: "pkill -f 'air' && air &"  # restart Go hot-reloader
      stages: ["implement"]

    - command: "systemctl --user restart my-dev-server"
      stages: ["implement"]
```

### 10. Artifact Management

Archive or backup generated files:

```yaml
hooks:
  on_command_complete:
    - command: |
        TIMESTAMP=$(date +%Y%m%d_%H%M%S)
        cp specs/$AUTOSPEC_SPEC/tasks.yaml ~/.autospec/backups/$AUTOSPEC_SPEC-$TIMESTAMP.yaml
      commands: ["tasks"]

    - command: "autospec export $AUTOSPEC_SPEC -o /backups/"
      commands: ["implement"]
      only_on_success: true
```

### 11. Security Scanning

Run security checks after implementation:

```yaml
hooks:
  on_stage_complete:
    - command: "gosec ./..."
      stages: ["implement"]

    - command: "trivy fs ."
      stages: ["implement"]

    - command: "gitleaks detect --source ."
      stages: ["implement"]
```

### 12. Metrics & Logging

Track execution metrics:

```yaml
hooks:
  on_command_complete:
    - command: |
        echo "$AUTOSPEC_COMMAND,$AUTOSPEC_SPEC,$AUTOSPEC_SUCCESS,$AUTOSPEC_DURATION,$(date -Iseconds)" \
          >> ~/.autospec/metrics.csv

    - command: |
        # Send to InfluxDB/Prometheus
        curl -i -XPOST 'http://localhost:8086/write?db=devmetrics' \
          --data-binary "autospec,command=$AUTOSPEC_COMMAND,spec=$AUTOSPEC_SPEC duration=$AUTOSPEC_DURATION"
```

---

## Implementation Approach

### Option A: Shell Hook Execution (Simple)

Add a `hooks` field to config and execute commands via `os/exec`:

```go
// internal/notify/hook_executor.go
type HookConfig struct {
    Command           string   `yaml:"command"`
    OnlyOnSuccess     bool     `yaml:"only_on_success"`
    Stages            []string `yaml:"stages"`
    Commands          []string `yaml:"commands"`
    BlockOnFailure    bool     `yaml:"block_on_failure"`
    RequireConfirm    bool     `yaml:"require_confirmation"`
}

func (h *Handler) executeHooks(hookType string, ctx HookContext) error {
    hooks := h.config.Hooks[hookType]
    for _, hook := range hooks {
        if !hook.shouldRun(ctx) {
            continue
        }
        cmd := expandEnvVars(hook.Command, ctx)
        if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
            if hook.BlockOnFailure {
                return fmt.Errorf("hook failed: %w", err)
            }
        }
    }
    return nil
}
```

### Option B: Plugin System (Advanced)

Use Go plugins for more complex hooks:

```go
// ~/.config/autospec/plugins/lint-on-complete.so
type Hook interface {
    Name() string
    OnCommandComplete(ctx HookContext) error
    OnStageComplete(ctx HookContext) error
    OnError(ctx HookContext) error
}
```

### Option C: Event System (Most Flexible)

Emit events that external tools can subscribe to:

```bash
# autospec emits to Unix socket or named pipe
autospec implement 2>&1 | tee >(
  while read line; do
    if [[ "$line" =~ "Stage complete:" ]]; then
      make test
    fi
  done
)
```

---

## Comparison with Other Tools

| Tool | Hook Mechanism | Key Features |
|------|----------------|--------------|
| **Git** | `.git/hooks/` scripts | pre-commit, post-merge, etc. |
| **Husky** | npm package + hooks | Easy setup, lint-staged integration |
| **pre-commit** | YAML config + plugins | Multi-language, large plugin ecosystem |
| **Make** | Target dependencies | Build-system-native |
| **Task (taskfile)** | YAML + deps | Modern Make alternative |
| **Just** | Justfile recipes | Simple task runner |
| **Lefthook** | YAML + parallel | Fast, parallel hook runner |

autospec hooks would complement these by triggering at **workflow stage boundaries** rather than git operations.

---

## Recommended Implementation Priority

1. **Phase 1: Shell Command Hooks**
   - Add `hooks` config section
   - Execute shell commands on events
   - Pass environment variables for context
   - Low effort, high value

2. **Phase 2: Conditional Execution**
   - Filter by stage, command, success/failure
   - Add `block_on_failure` for quality gates
   - Add `require_confirmation` for destructive ops

3. **Phase 3: Advanced Features**
   - Hook timeout configuration
   - Parallel hook execution
   - Hook output capture and logging
   - Plugin system for complex hooks

---

## Related Existing Features

- **Feature #18 in feature-ideas.md**: "Webhook/Event System" - aligns with this proposal
- **Notification system**: Already has the hook points, just needs command execution

## References

- [Git Hooks Best Practices](https://www.geeksforgeeks.org/git/customizing-git-hooks-for-workflow-automation/)
- [Pre-commit Framework](https://pre-commit.com/)
- [Husky](https://typicode.github.io/husky/)
- [Lefthook](https://github.com/evilmartians/lefthook)
- [12-Factor CLI Apps](https://medium.com/@jdxcode/12-factor-cli-apps-dd3c227a0e46)
