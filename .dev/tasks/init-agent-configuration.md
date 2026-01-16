# Agent-Aware Init Configuration

**Status:** Proposed
**Priority:** High
**Related:** `autospec init`, `internal/cliagent`, `internal/claude`
**Supersedes:** Extends `claude-settings-validation.md`

---

## Problem Statement

Currently `autospec init` automatically configures Claude Code settings (`.claude/settings.local.json`) without asking the user which agent(s) they intend to use. This is problematic because:

1. **Wrong assumption**: Not all users use Claude Code - they may use Gemini CLI, Cline, Codex, etc.
2. **Incomplete permissions**: Current implementation only adds `Bash(autospec:*)` but Claude also needs write/edit permissions for `.autospec/**` and `specs/**` to work effectively
3. **No persistence**: User's agent preference isn't stored, so future inits don't remember their choice

---

## Proposed Solution

### 1. Agent Selection Prompt

Add an interactive prompt during `autospec init` to select which agent(s) to configure:

```
$ autospec init

Select agent(s) to configure (space=toggle, enter=confirm):
  [x] claude (Recommended)
  [ ] gemini
  [ ] cline
  [ ] codex
  [ ] goose
  [ ] opencode
  [ ] custom / none

✓ Claude Code: configured .claude/settings.local.json
✓ Commands: 5 installed
✓ Config: created
...
```

### 2. Config Persistence

Store selected agents in `config.yml` for future inits:

```yaml
# Agent settings
agent_preset: claude              # Primary agent for workflow execution
default_agents: ["claude"]        # Pre-selected agents for init prompts
```

When `default_agents` is set, future `autospec init` runs pre-select those agents in the prompt.

### 3. Configurator Interface

Add optional `Configurator` interface to `internal/cliagent`:

```go
// Configurator is optionally implemented by agents that need project-level setup.
// Not all agents need this - only those with settings files (like Claude Code).
type Configurator interface {
    // ConfigureProject sets up agent-specific settings files for a project.
    // specsDir is the user's configured specs directory (default: "specs").
    // Returns nil if no configuration is needed or already configured.
    ConfigureProject(projectDir, specsDir string) error
}
```

### 4. Claude Configurator Implementation

Claude agent implements `Configurator` to set up `.claude/settings.local.json`:

```go
func (c *Claude) ConfigureProject(projectDir, specsDir string) error {
    settings, err := claude.Load(projectDir)
    if err != nil {
        return fmt.Errorf("loading claude settings: %w", err)
    }

    // Check deny list first
    requiredPerms := c.requiredPermissions(specsDir)
    for _, perm := range requiredPerms {
        if settings.CheckDenyList(perm) {
            return fmt.Errorf("permission %s is in deny list", perm)
        }
    }

    // Add missing permissions
    added := false
    for _, perm := range requiredPerms {
        if !settings.HasPermission(perm) {
            settings.AddPermission(perm)
            added = true
        }
    }

    if added {
        return settings.Save()
    }
    return nil
}

func (c *Claude) requiredPermissions(specsDir string) []string {
    return []string{
        "Bash(autospec:*)",
        "Write(.autospec/**)",
        "Edit(.autospec/**)",
        fmt.Sprintf("Write(%s/**)", specsDir),
        fmt.Sprintf("Edit(%s/**)", specsDir),
    }
}
```

---

## Implementation Plan

### Phase 1: Add Configurator Interface

**Files:**
- `internal/cliagent/configurator.go` - New interface definition

```go
package cliagent

// Configurator is optionally implemented by agents that need project-level setup.
type Configurator interface {
    // ConfigureProject sets up agent-specific settings for this project.
    // projectDir is the project root (usually ".").
    // specsDir is the configured specs directory (e.g., "specs").
    ConfigureProject(projectDir, specsDir string) error
}

// Configure attempts to configure an agent for the project.
// Returns nil if agent doesn't implement Configurator.
func Configure(agent Agent, projectDir, specsDir string) error {
    if configurator, ok := agent.(Configurator); ok {
        return configurator.ConfigureProject(projectDir, specsDir)
    }
    return nil
}
```

### Phase 2: Claude Implements Configurator

**Files:**
- `internal/cliagent/claude.go` - Add ConfigureProject method
- `internal/claude/settings.go` - Extend with new permission helpers

Update `internal/claude/settings.go`:
```go
// RequiredPermissions returns all permissions needed for autospec.
// specsDir is the configured specs directory.
func RequiredPermissions(specsDir string) []string {
    return []string{
        "Bash(autospec:*)",
        "Write(.autospec/**)",
        "Edit(.autospec/**)",
        fmt.Sprintf("Write(%s/**)", specsDir),
        fmt.Sprintf("Edit(%s/**)", specsDir),
    }
}
```

### Phase 3: Add Config Fields

**Files:**
- `internal/config/config.go` - Add `DefaultAgents` field
- `internal/config/defaults.go` - Add default value

```go
// Configuration struct addition
type Configuration struct {
    // ... existing fields ...

    // DefaultAgents are pre-selected in init prompts.
    // Stored from previous init selections.
    DefaultAgents []string `koanf:"default_agents"`
}
```

Default config template addition:
```yaml
# Pre-selected agents for init prompts (saved from previous selection)
default_agents: []
```

### Phase 4: Update Init Command

**Files:**
- `internal/cli/config/init_cmd.go` - Add agent selection prompt

```go
func runInit(cmd *cobra.Command, args []string) error {
    // ... existing setup ...

    // Load config to get defaults
    cfg, _ := config.Load("")

    // Prompt for agent selection
    selectedAgents := promptAgentSelection(cmd, cfg.DefaultAgents)

    // Configure selected agents
    for _, agentName := range selectedAgents {
        if err := configureAgent(out, agentName, cfg.SpecsDir); err != nil {
            fmt.Fprintf(out, "⚠ %s: %v\n", agentName, err)
        }
    }

    // Save selected agents to config for next time
    if len(selectedAgents) > 0 {
        saveDefaultAgents(selectedAgents)
    }

    // ... rest of existing code ...
}

func promptAgentSelection(cmd *cobra.Command, defaults []string) []string {
    agents := cliagent.List() // ["claude", "cline", "codex", "gemini", "goose", "opencode"]

    // Build options with defaults pre-selected
    // Use survey or similar library for multi-select
    // Return selected agent names
}

func configureAgent(out io.Writer, agentName, specsDir string) error {
    agent := cliagent.Get(agentName)
    if agent == nil {
        return fmt.Errorf("unknown agent: %s", agentName)
    }

    if err := cliagent.Configure(agent, ".", specsDir); err != nil {
        return err
    }

    fmt.Fprintf(out, "✓ %s: configured\n", agentName)
    return nil
}
```

### Phase 5: Remove Auto-Configure

**Files:**
- `internal/cli/config/init_cmd.go` - Remove `configureClaudeSettings` auto-call

Current code to remove/modify:
```go
// This runs unconditionally - WRONG
configureClaudeSettings(out, ".")
```

Should only run if user selected Claude in the agent prompt.

---

## UX Flow

### First-time init (no config)

```
$ autospec init

Select agent(s) to configure (space=toggle, enter=confirm):
❯ [x] claude (Recommended)
  [ ] cline
  [ ] codex
  [ ] gemini
  [ ] goose
  [ ] opencode

✓ Claude Code: configured .claude/settings.local.json
  - Bash(autospec:*)
  - Write(.autospec/**)
  - Edit(.autospec/**)
  - Write(specs/**)
  - Edit(specs/**)
✓ Commands: 8 installed → .claude/commands/
✓ Config: created at ~/.config/autospec/config.yml
  - default_agents: ["claude"] (saved for next time)
...
```

### Subsequent init (has config with default_agents)

```
$ autospec init

Select agent(s) to configure (space=toggle, enter=confirm):
❯ [x] claude (from config)
  [ ] cline
  ...

✓ Claude Code: already configured
✓ Commands: up to date
✓ Config: exists
...
```

### User selects no agents

```
$ autospec init

Select agent(s) to configure (space=toggle, enter=confirm):
❯ [ ] claude
  [ ] cline
  ...
  [x] Skip agent configuration

✓ Commands: 8 installed
✓ Config: created
⚠ No agents configured. You may need to manually configure your agent's permissions.
...
```

---

## Permissions Added by Claude Configurator

| Permission | Purpose |
|------------|---------|
| `Bash(autospec:*)` | Run any autospec CLI command |
| `Write(.autospec/**)` | Create files in .autospec/ (constitution, memory, config) |
| `Edit(.autospec/**)` | Modify files in .autospec/ |
| `Write(specs/**)` | Create spec files (spec.yaml, plan.yaml, tasks.yaml) |
| `Edit(specs/**)` | Modify spec files |

**Note:** Read permissions are not needed - Claude Code can read by default.

---

## Testing Strategy

### Unit Tests

1. **Configurator interface:**
   - Agent without Configurator returns nil
   - Claude implements Configurator correctly
   - Permissions use correct specsDir

2. **Config persistence:**
   - DefaultAgents saved to config
   - DefaultAgents loaded from config
   - Empty defaults handled

3. **Permission generation:**
   - RequiredPermissions includes all needed perms
   - Custom specsDir reflected in permissions

### Integration Tests

1. **Init with agent selection:**
   - Selected agents get configured
   - Unselected agents not touched
   - DefaultAgents saved

2. **Init with existing config:**
   - DefaultAgents pre-selected
   - Already-configured agents shown as such

---

## Files to Create/Modify

| File | Action |
|------|--------|
| `internal/cliagent/configurator.go` | Create - Configurator interface |
| `internal/cliagent/claude.go` | Modify - Implement Configurator |
| `internal/claude/settings.go` | Modify - Add RequiredPermissions helper |
| `internal/config/config.go` | Modify - Add DefaultAgents field |
| `internal/config/defaults.go` | Modify - Add default_agents template |
| `internal/cli/config/init_cmd.go` | Modify - Add agent selection, remove auto-configure |

---

## Dependencies

- Multi-select prompt library (e.g., `github.com/AlecAivazis/survey/v2` or `github.com/charmbracelet/huh`)
- Existing `internal/cliagent` package (merged from 062-agent-abstraction)
- Existing `internal/claude` package

---

## Open Questions

1. **Should we support `--skip-agent-config` flag?**
   - For CI/scripting where prompts aren't wanted
   - Recommendation: Yes, add `--no-agents` flag

2. **What about agents that don't need configuration?**
   - Gemini, Cline, etc. may not have settings files
   - Recommendation: Only show "configured" for agents that implement Configurator

3. **Should we validate agents are installed before configuring?**
   - Could skip uninstalled agents
   - Recommendation: Configure anyway - user might install later

---

## References

- [Claude Code Settings](https://docs.anthropic.com/en/docs/claude-code/settings)
- [claude-settings-validation.md](.dev/tasks/claude-settings-validation.md) - Original simpler implementation
- [docs/agents.md](../../docs/agents.md) - Agent abstraction documentation
