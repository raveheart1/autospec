# Agent Abstraction Architecture for Autospec

> Status: Design Document
> Created: 2025-12-19

## Overview

This document describes a Go architecture for abstracting CLI AI coding agents, enabling autospec to support multiple agents (Claude Code, Cline, Gemini, Codex, OpenCode, Goose) through a unified interface.

## Requirements for Automation

An agent **must support** these capabilities to work with autospec:

1. **Non-interactive prompt passing** - CLI arg like `-p`, `--message`, or subcommand
2. **Autonomous/YOLO mode** - Skip confirmations, no user input required
3. **Headless execution** - Works without TTY

---

## Package Structure

```
internal/
├── agent/                      # Agent abstraction layer
│   ├── agent.go               # Core interfaces
│   ├── base.go                # BaseAgent with shared implementation
│   ├── registry.go            # Agent registry & discovery
│   ├── custom.go              # CustomAgent ({{PROMPT}} templates)
│   ├── executor.go            # Shared execution helpers
│   │
│   ├── impl/                  # Concrete implementations
│   │   ├── claude.go
│   │   ├── cline.go
│   │   ├── gemini.go
│   │   ├── codex.go
│   │   ├── opencode.go
│   │   └── goose.go
│   │
│   └── output/                # Output parsing
│       ├── parser.go
│       ├── streamjson.go
│       └── text.go
│
├── workflow/
│   └── executor.go            # Refactored from claude.go to use agent.Agent
```

---

## Core Interfaces

### `internal/agent/agent.go`

```go
package agent

import (
    "context"
    "io"
    "os/exec"
    "time"
)

// Agent defines the contract for any CLI coding agent
type Agent interface {
    // Identity
    Name() string
    Version() (string, error)

    // Lifecycle
    Validate() error  // Check installed, API keys present

    // Execution
    BuildCommand(prompt string, opts ExecOptions) (*exec.Cmd, error)
    Execute(ctx context.Context, prompt string, opts ExecOptions) (*Result, error)

    // Discovery
    Capabilities() Caps
}

// Caps describes agent capabilities (self-describing)
type Caps struct {
    // CRITICAL: Automation requirements
    Automatable    bool           // Can run fully headless
    PromptDelivery PromptDelivery // How to pass the prompt
    AutonomousFlag string         // Skip-confirmation flag (e.g., "--dangerously-skip-permissions")
    AutonomousEnv  map[string]string // Env vars for autonomous mode (e.g., {"GOOSE_MODE": "auto"})

    // Features
    MCP         bool   // Model Context Protocol support
    LocalModels bool   // Ollama, local LLM support
    GitAware    bool   // Auto-commit, diff understanding
    Streaming   bool   // Stream output as it generates

    // Environment
    RequiredEnv   []string      // Required env vars (API keys)
    OptionalEnv   []string      // Optional env vars
    OutputFormats []OutputFmt   // Supported output formats
    DefaultModel  string        // Default model if applicable
}

// PromptDelivery describes how to pass prompts to the agent
type PromptDelivery struct {
    Method     string // "arg", "positional", "subcommand", "subcommand-arg"
    Flag       string // "-p", "exec", "run", etc.
    PromptFlag string // For subcommand-arg: flag after subcommand (e.g., "-t" for goose)
}

// CanAutomate validates automation requirements
func (c Caps) CanAutomate() bool {
    return c.Automatable && c.PromptDelivery.Method != ""
}

// BuildPromptArgs returns the args needed to pass a prompt
func (p PromptDelivery) BuildPromptArgs(prompt string) []string {
    switch p.Method {
    case "arg":
        return []string{p.Flag, prompt}
    case "positional":
        return []string{prompt}
    case "subcommand":
        return []string{p.Flag, prompt}
    case "subcommand-arg":
        return []string{p.Flag, p.PromptFlag, prompt}
    default:
        return []string{prompt}
    }
}

// ExecOptions configures a single execution
type ExecOptions struct {
    Model      string
    Format     OutputFmt
    Timeout    time.Duration
    WorkDir    string
    Autonomous bool          // Use autonomous/YOLO mode
    Verbose    bool
    ExtraArgs  []string
    Env        map[string]string
    Stdin      io.Reader
    Stdout     io.Writer
    Stderr     io.Writer
}

// Result captures execution outcome
type Result struct {
    ExitCode   int
    Stdout     string
    Stderr     string
    Duration   time.Duration
    Parsed     *ParsedOutput  // nil if parsing not supported
}

// ParsedOutput for agents that support structured output
type ParsedOutput struct {
    Messages    []Message
    ToolCalls   []ToolCall
    FilesEdited []string
    Summary     string
}

// Message represents a conversation turn
type Message struct {
    Role    string    // "user", "assistant", "system"
    Content string
    Time    time.Time
}

// ToolCall represents an agent action
type ToolCall struct {
    Name    string
    Args    map[string]any
    Result  string
    Success bool
}

// OutputFmt enum
type OutputFmt string

const (
    FmtText       OutputFmt = "text"
    FmtJSON       OutputFmt = "json"
    FmtStreamJSON OutputFmt = "stream-json"
)
```

---

## Base Implementation

### `internal/agent/base.go`

```go
package agent

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "strings"
    "time"
)

// BaseAgent provides shared functionality for all agents
type BaseAgent struct {
    name        string
    cmd         string   // CLI command (e.g., "claude", "cline")
    versionFlag string   // --version, -v, etc.
    caps        Caps
}

func (b *BaseAgent) Name() string { return b.name }

func (b *BaseAgent) Version() (string, error) {
    if b.versionFlag == "" {
        return "unknown", nil
    }
    out, err := exec.Command(b.cmd, b.versionFlag).Output()
    if err != nil {
        return "", fmt.Errorf("getting %s version: %w", b.name, err)
    }
    return strings.TrimSpace(string(out)), nil
}

func (b *BaseAgent) Validate() error {
    // Check command exists in PATH
    if _, err := exec.LookPath(b.cmd); err != nil {
        return fmt.Errorf("%s not found in PATH: %w", b.cmd, err)
    }

    // Check required env vars
    for _, env := range b.caps.RequiredEnv {
        if os.Getenv(env) == "" {
            return fmt.Errorf("required env var %s not set for %s", env, b.name)
        }
    }
    return nil
}

func (b *BaseAgent) Capabilities() Caps { return b.caps }

func (b *BaseAgent) BuildCommand(prompt string, opts ExecOptions) (*exec.Cmd, error) {
    var args []string

    // Add prompt args based on delivery method
    args = append(args, b.caps.PromptDelivery.BuildPromptArgs(prompt)...)

    // Add autonomous flag if requested and available
    if opts.Autonomous && b.caps.AutonomousFlag != "" {
        args = append(args, b.caps.AutonomousFlag)
    }

    // Add extra args
    args = append(args, opts.ExtraArgs...)

    cmd := exec.Command(b.cmd, args...)
    cmd.Dir = opts.WorkDir
    cmd.Env = b.buildEnv(opts)

    return cmd, nil
}

func (b *BaseAgent) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Result, error) {
    cmd, err := b.BuildCommand(prompt, opts)
    if err != nil {
        return nil, fmt.Errorf("building command: %w", err)
    }

    // Apply timeout
    if opts.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
        defer cancel()
    }

    // Set up I/O
    if opts.Stdout != nil {
        cmd.Stdout = opts.Stdout
    }
    if opts.Stderr != nil {
        cmd.Stderr = opts.Stderr
    }
    if opts.Stdin != nil {
        cmd.Stdin = opts.Stdin
    }

    start := time.Now()
    err = cmd.Run()
    duration := time.Since(start)

    result := &Result{
        Duration: duration,
    }

    if cmd.ProcessState != nil {
        result.ExitCode = cmd.ProcessState.ExitCode()
    }

    if err != nil {
        return result, fmt.Errorf("executing %s: %w", b.name, err)
    }

    return result, nil
}

func (b *BaseAgent) buildEnv(opts ExecOptions) []string {
    env := os.Environ()

    // Add autonomous env vars if requested
    if opts.Autonomous {
        for k, v := range b.caps.AutonomousEnv {
            env = append(env, fmt.Sprintf("%s=%s", k, v))
        }
    }

    // Add custom env vars
    for k, v := range opts.Env {
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }

    return env
}
```

---

## Registry

### `internal/agent/registry.go`

```go
package agent

import (
    "fmt"
    "sort"
    "sync"
)

// Registry manages available agents
type Registry struct {
    agents map[string]Agent
    mu     sync.RWMutex
}

// Default is the global agent registry
var Default = NewRegistry()

func NewRegistry() *Registry {
    return &Registry{agents: make(map[string]Agent)}
}

func (r *Registry) Register(a Agent) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.agents[a.Name()] = a
}

func (r *Registry) Get(name string) (Agent, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    if a, ok := r.agents[name]; ok {
        return a, nil
    }
    return nil, fmt.Errorf("agent %q not registered", name)
}

func (r *Registry) List() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    names := make([]string, 0, len(r.agents))
    for name := range r.agents {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}

// Available returns only installed and configured agents
func (r *Registry) Available() []Agent {
    r.mu.RLock()
    defer r.mu.RUnlock()
    var available []Agent
    for _, a := range r.agents {
        if a.Validate() == nil {
            available = append(available, a)
        }
    }
    return available
}

// Automatable returns only agents that support automation
func (r *Registry) Automatable() []Agent {
    r.mu.RLock()
    defer r.mu.RUnlock()
    var agents []Agent
    for _, a := range r.agents {
        if a.Capabilities().CanAutomate() {
            agents = append(agents, a)
        }
    }
    return agents
}

// Status represents health check result for an agent
type Status struct {
    Name      string
    Installed bool
    Version   string
    Valid     bool
    Errors    []string
}

// Doctor checks health of all registered agents
func (r *Registry) Doctor() []Status {
    r.mu.RLock()
    defer r.mu.RUnlock()

    statuses := make([]Status, 0, len(r.agents))
    for name, a := range r.agents {
        s := Status{Name: name}

        if v, err := a.Version(); err == nil {
            s.Installed = true
            s.Version = v
        } else {
            s.Errors = append(s.Errors, err.Error())
        }

        if err := a.Validate(); err == nil {
            s.Valid = true
        } else {
            s.Errors = append(s.Errors, err.Error())
        }

        statuses = append(statuses, s)
    }

    // Sort by name for consistent output
    sort.Slice(statuses, func(i, j int) bool {
        return statuses[i].Name < statuses[j].Name
    })

    return statuses
}
```

---

## Custom Agent (Template-Based)

### `internal/agent/custom.go`

```go
package agent

import (
    "context"
    "errors"
    "fmt"
    "os"
    "os/exec"
    "strings"
    "time"

    "github.com/google/shlex"
)

// CustomAgent supports arbitrary CLI tools via {{PROMPT}} templates
type CustomAgent struct {
    name     string
    template string // e.g., "aider --model sonnet --yes-always --message {{PROMPT}}"
    caps     Caps
}

// NewCustomAgent creates an agent from a command template
func NewCustomAgent(name, template string) (*CustomAgent, error) {
    if !strings.Contains(template, "{{PROMPT}}") {
        return nil, errors.New("template must contain {{PROMPT}} placeholder")
    }
    return &CustomAgent{
        name:     name,
        template: template,
        caps: Caps{
            Automatable:    true,
            PromptDelivery: PromptDelivery{Method: "template"},
        },
    }, nil
}

func (c *CustomAgent) Name() string             { return c.name }
func (c *CustomAgent) Version() (string, error) { return "custom", nil }
func (c *CustomAgent) Validate() error          { return nil }
func (c *CustomAgent) Capabilities() Caps       { return c.caps }

func (c *CustomAgent) BuildCommand(prompt string, opts ExecOptions) (*exec.Cmd, error) {
    // Expand template with quoted prompt
    cmdLine := strings.Replace(c.template, "{{PROMPT}}", shellQuote(prompt), 1)

    // Parse into command + args
    parts, err := shlex.Split(cmdLine)
    if err != nil {
        return nil, fmt.Errorf("parsing template: %w", err)
    }

    if len(parts) == 0 {
        return nil, errors.New("empty command after template expansion")
    }

    cmd := exec.Command(parts[0], parts[1:]...)
    cmd.Dir = opts.WorkDir
    cmd.Env = os.Environ()

    for k, v := range opts.Env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }

    return cmd, nil
}

func (c *CustomAgent) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Result, error) {
    cmd, err := c.BuildCommand(prompt, opts)
    if err != nil {
        return nil, err
    }

    if opts.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
        defer cancel()
    }

    if opts.Stdout != nil {
        cmd.Stdout = opts.Stdout
    }
    if opts.Stderr != nil {
        cmd.Stderr = opts.Stderr
    }
    if opts.Stdin != nil {
        cmd.Stdin = opts.Stdin
    }

    start := time.Now()
    err = cmd.Run()

    result := &Result{
        Duration: time.Since(start),
    }

    if cmd.ProcessState != nil {
        result.ExitCode = cmd.ProcessState.ExitCode()
    }

    return result, err
}

// shellQuote quotes a string for shell usage
func shellQuote(s string) string {
    if !strings.ContainsAny(s, " \t\n\"'`$\\") {
        return s
    }
    return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
```

---

## Concrete Implementations

### `internal/agent/impl/agents.go`

```go
package impl

import "github.com/ariel-frischer/autospec/internal/agent"

// NewClaude creates the Claude Code agent
func NewClaude() agent.Agent {
    return &agent.BaseAgent{
        name:        "claude",
        cmd:         "claude",
        versionFlag: "--version",
        caps: agent.Caps{
            Automatable: true,
            PromptDelivery: agent.PromptDelivery{
                Method: "arg",
                Flag:   "-p",
            },
            AutonomousFlag: "--dangerously-skip-permissions",
            MCP:            true,
            GitAware:       true,
            Streaming:      true,
            RequiredEnv:    []string{"ANTHROPIC_API_KEY"},
            OutputFormats:  []agent.OutputFmt{agent.FmtText, agent.FmtStreamJSON},
        },
    }
}

// NewCline creates the Cline agent
func NewCline() agent.Agent {
    return &agent.BaseAgent{
        name:        "cline",
        cmd:         "cline",
        versionFlag: "--version",
        caps: agent.Caps{
            Automatable: true,
            PromptDelivery: agent.PromptDelivery{
                Method: "positional",
            },
            AutonomousFlag: "-Y",  // YOLO mode
            GitAware:       true,
            LocalModels:    true,
            OutputFormats:  []agent.OutputFmt{agent.FmtText, agent.FmtJSON},
        },
    }
}

// NewGemini creates the Google Gemini CLI agent
func NewGemini() agent.Agent {
    return &agent.BaseAgent{
        name:        "gemini",
        cmd:         "gemini",
        versionFlag: "--version",
        caps: agent.Caps{
            Automatable: true,
            PromptDelivery: agent.PromptDelivery{
                Method: "arg",
                Flag:   "-p",
            },
            AutonomousFlag: "--yolo",
            MCP:            true,
            GitAware:       true,
            Streaming:      true,
            RequiredEnv:    []string{"GEMINI_API_KEY"},
            OutputFormats:  []agent.OutputFmt{agent.FmtText, agent.FmtJSON},
        },
    }
}

// NewCodex creates the OpenAI Codex CLI agent
func NewCodex() agent.Agent {
    return &agent.BaseAgent{
        name:        "codex",
        cmd:         "codex",
        versionFlag: "--version",
        caps: agent.Caps{
            Automatable: true,
            PromptDelivery: agent.PromptDelivery{
                Method: "subcommand",
                Flag:   "exec",
            },
            // exec mode is inherently autonomous
            MCP:         true,
            GitAware:    true,
            RequiredEnv: []string{"OPENAI_API_KEY"},
        },
    }
}

// NewOpenCode creates the OpenCode agent
func NewOpenCode() agent.Agent {
    return &agent.BaseAgent{
        name:        "opencode",
        cmd:         "opencode",
        versionFlag: "--version",
        caps: agent.Caps{
            Automatable: true,
            PromptDelivery: agent.PromptDelivery{
                Method: "subcommand",
                Flag:   "run",
            },
            // run mode is inherently non-interactive
            MCP:         true,
            LocalModels: true,
            GitAware:    true,
        },
    }
}

// NewGoose creates the Goose agent (Block/Linux Foundation)
func NewGoose() agent.Agent {
    return &agent.BaseAgent{
        name:        "goose",
        cmd:         "goose",
        versionFlag: "--version",
        caps: agent.Caps{
            Automatable: true,
            PromptDelivery: agent.PromptDelivery{
                Method:     "subcommand-arg",
                Flag:       "run",
                PromptFlag: "-t",
            },
            AutonomousFlag: "--no-session",
            AutonomousEnv:  map[string]string{"GOOSE_MODE": "auto"},
            MCP:            true,
            LocalModels:    true,
            GitAware:       true,
        },
    }
}

// RegisterAll registers all built-in agents with the default registry
func RegisterAll() {
    agent.Default.Register(NewClaude())
    agent.Default.Register(NewCline())
    agent.Default.Register(NewGemini())
    agent.Default.Register(NewCodex())
    agent.Default.Register(NewOpenCode())
    agent.Default.Register(NewGoose())
}

func init() {
    RegisterAll()
}
```

---

## Config Integration

### Updated `internal/config/config.go` fields

```go
type Config struct {
    // ... existing fields ...

    // Agent configuration (new)
    AgentPreset    string            `yaml:"agent_preset"`     // "claude", "cline", "gemini", etc.
    AgentCmd       string            `yaml:"agent_cmd"`        // Override command
    AgentArgs      []string          `yaml:"agent_args"`       // Override args
    CustomAgentCmd string            `yaml:"custom_agent_cmd"` // Template with {{PROMPT}}

    // Backward compatibility (deprecated)
    ClaudeCmd       string   `yaml:"claude_cmd"`        // Deprecated: use agent_cmd
    ClaudeArgs      []string `yaml:"claude_args"`       // Deprecated: use agent_args
    CustomClaudeCmd string   `yaml:"custom_claude_cmd"` // Deprecated: use custom_agent_cmd
}

// GetAgent returns the configured agent
func (c *Config) GetAgent() (agent.Agent, error) {
    // Priority: custom_agent_cmd > agent_preset > claude_cmd (backward compat)

    if c.CustomAgentCmd != "" {
        return agent.NewCustomAgent("custom", c.CustomAgentCmd)
    }

    // Backward compatibility
    if c.CustomClaudeCmd != "" {
        return agent.NewCustomAgent("custom", c.CustomClaudeCmd)
    }

    preset := c.AgentPreset
    if preset == "" {
        preset = "claude" // Default
    }

    return agent.Default.Get(preset)
}
```

---

## Usage Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/ariel-frischer/autospec/internal/agent"
    _ "github.com/ariel-frischer/autospec/internal/agent/impl" // Register agents
)

func main() {
    // Get an agent from the registry
    a, err := agent.Default.Get("claude")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    // Validate it's installed and configured
    if err := a.Validate(); err != nil {
        fmt.Fprintf(os.Stderr, "Agent not ready: %v\n", err)
        os.Exit(1)
    }

    // Execute a prompt
    result, err := a.Execute(context.Background(), "Fix the bug in auth.py", agent.ExecOptions{
        Autonomous: true,
        Timeout:    5 * time.Minute,
        WorkDir:    ".",
        Stdout:     os.Stdout,
        Stderr:     os.Stderr,
    })

    if err != nil {
        fmt.Fprintf(os.Stderr, "Execution failed: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Completed in %v with exit code %d\n", result.Duration, result.ExitCode)
}
```

---

## Tier 1 Agent Summary

| Agent | Command Pattern | Autonomous Mode |
|-------|-----------------|-----------------|
| **Claude Code** | `claude -p "task" --dangerously-skip-permissions` | Flag |
| **Cline** | `cline -Y "task"` | Flag |
| **Gemini CLI** | `gemini -p "task" --yolo` | Flag |
| **Codex CLI** | `codex exec "task"` | Inherent |
| **OpenCode** | `opencode run "task"` | Inherent |
| **Goose** | `GOOSE_MODE=auto goose run --no-session -t "task"` | Env + Flag |

---

## Migration Path

1. Create `internal/agent/` package with interfaces
2. Implement `BaseAgent` and `CustomAgent`
3. Add concrete implementations for Tier 1 agents
4. Create `Registry` for agent discovery
5. Refactor `internal/workflow/claude.go` to use `agent.Agent` interface
6. Add `agent_preset` config option (keep `claude_*` for backward compat)
7. Update `doctor` command to check all registered agents
8. Add `--agent` CLI flag for runtime selection
