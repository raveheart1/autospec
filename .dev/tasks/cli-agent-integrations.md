# CLI Agent Integrations for Autospec

> Research completed: 2025-12-19
> Status: Research / Planning

## Overview

Autospec currently supports Claude Code as its primary CLI agent. This document explores other CLI-based AI coding agents that could be integrated, enabling users to choose their preferred agent or use local/self-hosted models.

## Popularity Rankings (GitHub Stars, Dec 2025)

| Rank | Agent | GitHub Stars | Install | Primary Use Case |
|------|-------|--------------|---------|------------------|
| 1 | **Gemini CLI** | ~87k | `npm install -g @google/gemini-cli` | General AI CLI (coding + more) |
| 2 | **GPT Engineer** | ~54.6k | `pip install gpt-engineer` | Greenfield app generation |
| 3 | **Cline** | ~48k | `npm install -g cline` | Autonomous coding agent |
| 4 | **Claude Code** | ~40.7k | `npm install -g @anthropic-ai/claude-code` | Anthropic's official CLI |
| 5 | **Aider** | ~12.9k | `pip install aider-chat` | Git-aware pair programmer |
| 6 | **OpenCode** | Rising | `npm install -g opencode-ai` | TUI + LSP-enabled agent |
| 7 | **ForgeCode** | Rising | `npx forgecode@latest` | Zero-config CLI agent |
| 8 | **Goose** | Growing | From GitHub | Linux Foundation backed |
| 9 | **Qwen CLI** | New (Jul 2025) | `npm install -g @qwen-code/qwen-code` | Alibaba's Qwen3-Coder |
| 10 | **Grok CLI** | New | From GitHub | xAI's Grok models |

---

## Current Integration Architecture

Autospec uses a flexible executor pattern in `internal/workflow/claude.go`:

```go
type ClaudeExecutor struct {
    ClaudeCmd       string    // e.g., "claude", "aider", "gemini"
    ClaudeArgs      []string  // e.g., ["-p", "--verbose"]
    CustomClaudeCmd string    // e.g., "aider --model sonnet {{PROMPT}}"
    Timeout         int
}
```

**Key insight**: The `{{PROMPT}}` placeholder in `custom_claude_cmd` already enables arbitrary CLI tool integration.

---

## CLI Agents Research

### Tier 1: Production-Ready Agentic CLI Tools (High Priority)

These are mature CLI tools with **autonomous/YOLO mode** support - required for autospec automation.

**Requirements for Tier 1:**
- Non-interactive prompt passing (CLI arg like `-p`, `--message`, or subcommand)
- Autonomous/YOLO mode (skip confirmations, no user input required)
- Headless execution (works without TTY)

---

#### 1. Claude Code (Anthropic) ⭐ Current Default

| Attribute | Value |
|-----------|-------|
| **Install** | `npm install -g @anthropic-ai/claude-code` |
| **Command** | `claude` |
| **GitHub Stars** | ~40.7k |
| **License** | Proprietary (Anthropic) |
| **Models** | Claude Opus 4, Sonnet, etc. |
| **Maturity** | Stable (GA May 2025) |

**CLI Usage Patterns**:
```bash
# Interactive mode
claude

# With prompt (non-interactive)
claude -p "fix the bug in auth.py"

# Skip permissions (autonomous)
claude -p --dangerously-skip-permissions "refactor module"

# With output format
claude -p --output-format stream-json "task"
```

**Autospec Config (Current)**:
```yaml
claude_cmd: claude
claude_args:
  - -p
  - --dangerously-skip-permissions
  - --verbose
  - --output-format
  - stream-json
```

**Repo**: https://github.com/anthropics/claude-code

---

#### 2. Cline CLI

| Attribute | Value |
|-----------|-------|
| **Install** | `npm install -g cline` or `npx cline` |
| **Command** | `cline` |
| **GitHub Stars** | ~48k |
| **License** | Apache 2.0 (Open Source) |
| **Models** | OpenAI, Anthropic, local models |
| **Maturity** | Stable |

**CLI Usage Patterns**:
```bash
# Interactive mode
cline "implement user authentication"

# One-shot mode (autonomous)
cline -O "fix failing tests"
cline --one-shot "run tests and fix"

# YOLO mode (fully autonomous)
cline -Y "refactor module"

# Verbose output
cline -V "analyze dependencies"

# JSON output (for scripts/CI)
cline -F json "task description"
```

**Autospec Config Example**:
```yaml
claude_cmd: cline
claude_args:
  - -O  # One-shot mode
```

**Custom Command Example**:
```yaml
custom_claude_cmd: "cline -O {{PROMPT}}"
```

**Strengths**:
- Agentic multi-step execution
- Terminal command awareness
- Checkpoint system (beyond Git)
- Good for test-driven development

**Docs**: https://docs.cline.bot/cline-cli/cli-reference

---

#### 3. Google Gemini CLI

| Attribute | Value |
|-----------|-------|
| **Install** | `npm install -g @google/gemini-cli` |
| **Command** | `gemini` |
| **GitHub Stars** | ~87k |
| **License** | Apache 2.0 (Open Source) |
| **Models** | Gemini 2.5 Pro (1M token context) |
| **Maturity** | Stable (launched June 2025) |
| **Autonomous Mode** | `--yolo` or `-y` |

**CLI Usage Patterns**:
```bash
# Interactive mode
gemini

# Non-interactive with prompt
gemini -p "analyze this codebase"

# YOLO mode (fully autonomous, no confirmations)
gemini -p "refactor module" --yolo
gemini -y "fix the bug"

# Explicit approval mode
gemini --approval-mode yolo -p "task"

# With JSON output (for scripts/CI)
gemini -p "generate script" --yolo --output-format json

# Middle-ground: auto-approve edits only
gemini --approval-mode auto_edit -p "task"
```

**Autospec Config Example**:
```yaml
claude_cmd: gemini
claude_args:
  - -p
  - --yolo
```

**Custom Command Example**:
```yaml
custom_claude_cmd: "gemini -p {{PROMPT}} --yolo"
```

**Strengths**:
- 1M token context window
- Free tier: 60 req/min, 1000 req/day
- Full YOLO mode for automation
- Built-in Google Search grounding
- MCP integration
- Custom slash commands via `.toml`

**Repo**: https://github.com/google-gemini/gemini-cli
**Docs**: https://geminicli.com/docs/cli/

---

#### 4. OpenAI Codex CLI

| Attribute | Value |
|-----------|-------|
| **Install** | Via OpenAI developer tools |
| **Command** | `codex` |
| **License** | Proprietary (OpenAI) |
| **Models** | OpenAI models only (GPT-4o, GPT-5-Codex) |
| **Maturity** | Stable, polished UX |

**CLI Usage Patterns**:
```bash
# Interactive mode
codex

# Quick prompt (non-interactive)
codex "explain this codebase"

# With specific path
codex --path ./src "refactor this module"

# Exec mode (non-interactive, for scripts/CI)
codex exec "fix the CI failure"

# Code review
codex review --base main
```

**Autospec Config Example**:
```yaml
claude_cmd: codex
claude_args:
  - exec
```

**Custom Command Example**:
```yaml
custom_claude_cmd: "codex exec {{PROMPT}}"
```

**Access Modes**:
- `auto` (default): Read/edit/run in working directory
- `read-only`: Consultative only
- `full-access`: Machine-wide access (use cautiously)

**Strengths**:
- Polished UX
- Strong code quality benchmarks
- MCP integration support
- Image/screenshot input support

**Docs**: https://developers.openai.com/codex/cli/features/

---

#### 5. OpenCode

| Attribute | Value |
|-----------|-------|
| **Install** | `npm install -g opencode-ai@latest` |
| **Command** | `opencode` |
| **License** | Open Source |
| **Models** | Claude, GPT, Gemini, local models |
| **Maturity** | Active development (2025) |
| **Autonomous Mode** | `run` subcommand (inherently non-interactive) |

**CLI Usage Patterns**:
```bash
# Interactive TUI mode (default)
opencode

# Non-interactive mode (for automation)
opencode run "fix the bug in auth.py"
opencode run "refactor this module"

# Create custom agent
opencode agent create

# Manage agents
opencode agent list
```

**Environment Variables**:
```bash
ANTHROPIC_API_KEY=...   # Claude
OPENAI_API_KEY=...      # OpenAI
GEMINI_API_KEY=...      # Google
LOCAL_ENDPOINT=...      # Self-hosted
```

**Autospec Config Example**:
```yaml
claude_cmd: opencode
claude_args:
  - run
```

**Custom Command Example**:
```yaml
custom_claude_cmd: "opencode run {{PROMPT}}"
```

**Strengths**:
- `run` subcommand for headless automation
- TUI + CLI hybrid interface
- LSP-enabled (true refactors, rename-symbol)
- Multi-session support
- MCP integration
- MGrep for fast project search (4x faster)
- Local model support

**Repo**: https://github.com/sst/opencode
**Docs**: https://opencode.ai/docs/cli/

---

#### 6. Goose (Block/Linux Foundation)

| Attribute | Value |
|-----------|-------|
| **Install** | From GitHub releases or `brew install goose` |
| **Command** | `goose` |
| **License** | Apache 2.0 (Open Source) |
| **Models** | Any LLM provider (configurable) |
| **Maturity** | Stable (Linux Foundation backed) |
| **Autonomous Mode** | `GOOSE_MODE=auto` + `--no-session` |

**CLI Usage Patterns**:
```bash
# Interactive session
goose

# Non-interactive headless mode (for automation)
GOOSE_MODE=auto goose run --no-session -t "fix the bug"

# With developer extensions
GOOSE_MODE=auto goose run --no-session --with-builtin developer -t "refactor module"

# Switch to auto mode mid-session
/mode auto
```

**Permission Levels**:
| Mode | Autonomy | Use Case |
|------|----------|----------|
| `auto` (Autonomous) | Full (no approvals) | Automation, headless |
| `smart_approval` | Risk-based | Balanced oversight |
| `manual_approval` | Every action confirmed | High caution |
| `chat_only` | No tools/changes | Pure conversation |

**Autospec Config Example**:
```yaml
claude_cmd: goose
claude_args:
  - run
  - --no-session
  - -t
```

**Custom Command Example**:
```yaml
custom_claude_cmd: "GOOSE_MODE=auto goose run --no-session -t {{PROMPT}}"
```

**Strengths**:
- Full autonomous mode for CI/CD
- Local-first, on-machine agent
- MCP integration
- AGENTS.md conventions support
- Linux Foundation backed (Agentic AI Foundation)
- Configurable permission levels

**Repo**: https://github.com/block/goose
**Docs**: https://block.github.io/goose/docs/

---

### Tier 2: Emerging CLI Agents (Medium Priority)

These are newer or more specialized tools with growing adoption.

---

#### 7. ForgeCode (Needs Verification)

| Attribute | Value |
|-----------|-------|
| **Install** | `npx forgecode@latest` (zero-config) |
| **Command** | `npx forgecode@latest` or `forge` |
| **License** | Open Source |
| **Models** | OpenAI, Anthropic (via API keys) |
| **Maturity** | Active development |

**CLI Usage Patterns**:
```bash
# Run directly (recommended)
npx forgecode@latest

# In project directory
cd /path/to/project && npx forgecode@latest

# With path
npx forgecode@latest /path/to/project

# Global install
npm install -g forgecode@latest
forge
```

**Configuration**: Add `FORGE_KEY` or provider keys to `.env`

**Autospec Config Example**:
```yaml
custom_claude_cmd: "npx forgecode@latest --message {{PROMPT}}"
```

**Strengths**:
- Zero-config (npx)
- Terminal-first design
- Codebase understanding

**Repo**: https://github.com/antinomyhq/forge

---

#### 8. Qwen CLI (Alibaba) (Needs Verification)

| Attribute | Value |
|-----------|-------|
| **Install** | `npm install -g @qwen-code/qwen-code` |
| **Command** | `qwen` |
| **License** | Apache 2.0 (Open Source) |
| **Models** | Qwen3-Coder (up to 1M context) |
| **Maturity** | New (July 2025) |

**CLI Usage Patterns**:
```bash
# Interactive mode
qwen

# Version check
qwen --version
```

**Environment Variables**:
```bash
OPENAI_API_KEY="your_key"
OPENAI_BASE_URL="https://dashscope-intl.aliyuncs.com/compatible-mode/v1"
OPENAI_MODEL="qwen3-coder-plus"
```

**Autospec Config Example**:
```yaml
claude_cmd: qwen
claude_args: []
```

**Strengths**:
- 1M token context support
- Tops Terminal-Bench (37.5% accuracy)
- Vision model support
- OpenAI SDK compatible
- Works with Dashscope, OpenRouter, local models

**Repo**: https://github.com/QwenLM/qwen-code

---

#### 9. Grok CLI (xAI) (Needs Verification)

| Attribute | Value |
|-----------|-------|
| **Install** | From GitHub (Python or Node versions) |
| **Command** | `grok_cli` or `grok` |
| **License** | Open Source |
| **Models** | Grok-3, Grok-4, grok-code-fast-1 |
| **Maturity** | Active development |

**CLI Usage Patterns**:
```bash
# Python version
grok_cli --api-key YOUR_KEY

# Node/TUI version
grok
```

**Environment Variables**:
```bash
GROK_API_KEY=grk-...
# or
COMPOSIO_API_KEY=grk-...
```

**Autospec Config Example**:
```yaml
custom_claude_cmd: "grok_cli --api-key $GROK_API_KEY {{PROMPT}}"
```

**Strengths**:
- xAI's latest Grok models
- TUI interface (Ink-based)
- View/create/edit files
- Shell command execution
- GitHub Actions integration
- grok-code-fast-1 for speed

**Repo**: https://github.com/superagent-ai/grok-cli

---

#### 10. Rovo Dev CLI (Atlassian)

| Attribute | Value |
|-----------|-------|
| **Install** | Via Atlassian CLI (ACLI) |
| **Command** | `acli rovodev run` |
| **License** | Proprietary (Atlassian) |
| **Models** | Atlassian AI |
| **Maturity** | Enterprise-ready |

**CLI Usage Patterns**:
```bash
# Interactive mode
acli rovodev run

# Single command
acli rovodev run "summarize this repo"

# Shadow mode (temp workspace)
acli rovodev run --shadow

# YOLO mode (no confirmations)
acli rovodev run --yolo

# Server mode
acli rovodev serve 8080
```

**Autospec Config Example**:
```yaml
custom_claude_cmd: "acli rovodev run {{PROMPT}}"
```

**Strengths**:
- Jira/Confluence integration
- Enterprise permissions & credits
- MCP server support
- Shadow mode for safe testing
- Can update Jira issues, create Confluence docs

**Docs**: https://support.atlassian.com/rovo/docs/rovo-dev-cli-commands/

---

#### 11. GPT Engineer

| Attribute | Value |
|-----------|-------|
| **Install** | `pip install gpt-engineer` |
| **Command** | `gpte` |
| **GitHub Stars** | ~54.6k |
| **License** | MIT (Open Source) |
| **Models** | OpenAI (GPT-4, vision models) |
| **Maturity** | Stable |

**CLI Usage Patterns**:
```bash
# Generate from prompt file
gpte projects/my-project

# Improve existing code
gpte projects/my-project -i

# With vision model
gpte projects/example-vision gpt-4-vision-preview \
  --prompt_file prompt/text \
  --image_directory prompt/images -i
```

**Autospec Config Example**:
```yaml
claude_cmd: gpte
claude_args:
  - -i  # Improve mode
```

**Strengths**:
- Full codebase generation from prompts
- Improve mode for existing code
- Vision model support
- Custom preprompts/agent identity

**Repo**: https://github.com/AntonOsika/gpt-engineer

---

#### 12. Amazon Q Developer CLI

| Attribute | Value |
|-----------|-------|
| **Install** | Download from AWS |
| **Command** | `q` |
| **License** | Proprietary (AWS) |
| **Models** | AWS Bedrock (Claude 3.7 Sonnet) |
| **Maturity** | Stable in AWS ecosystem |

**CLI Usage Patterns**:
```bash
# Interactive chat
q chat

# Natural language to shell command
q translate "create an S3 bucket named my-bucket"

# Within chat session
> Write a script to count down from 10
```

**Autospec Config Example**:
```yaml
claude_cmd: q
claude_args:
  - chat
```

**Strengths**:
- Deep AWS integration
- IAM Identity Center support
- Can query AWS resources
- Reads/writes local files

**Docs**: https://docs.aws.amazon.com/amazonq/latest/qdeveloper-ug/command-line.html

---

### Tier 3: Specialized / Local-First Agents

---

#### 14. Open Interpreter

| Attribute | Value |
|-----------|-------|
| **Install** | `pip install open-interpreter` |
| **Command** | `interpreter` |
| **License** | AGPL-3.0 (Open Source) |
| **Models** | OpenAI, Anthropic, Ollama (local) |
| **Maturity** | Stable |

**CLI Usage Patterns**:
```bash
# Interactive mode
interpreter

# With specific model
interpreter --model gpt-4o
interpreter --local  # For Ollama

# Safe mode (review before execution)
interpreter --safe

# Non-interactive script
interpreter --script my_task.py
```

**Autospec Config Example**:
```yaml
claude_cmd: interpreter
claude_args:
  - --model
  - gpt-4o
```

**Strengths**:
- Code execution in real-time
- Multi-language support (Python, JS, Shell)
- Local model support via Ollama
- Safe mode for code review

**Repo**: https://github.com/openinterpreter/open-interpreter

---

#### 15. Ollamacode CLI (Local Models)

| Attribute | Value |
|-----------|-------|
| **Install** | From GitHub |
| **Command** | `ollamacode` |
| **License** | Open Source |
| **Models** | Any Ollama model (fully local) |
| **Maturity** | Active development |

**CLI Usage Patterns**:
```bash
# With local model
ollamacode --model qwen2.5-coder:7b-instruct

# Project-aware
ollamacode --source-code-path /path/to/repo
```

**Autospec Config Example**:
```yaml
custom_claude_cmd: "ollamacode --model codestral {{PROMPT}}"
```

**Strengths**:
- Fully local (no API calls, no data leaves machine)
- Direct tool execution (files, shell, git)
- Permission system
- Multi-language support

**Repo**: https://github.com/tooyipjee/ollamacode_cli

---

#### 16. Mentat

| Attribute | Value |
|-----------|-------|
| **Install** | From GitHub/pip |
| **Command** | `mentat` |
| **License** | Open Source |
| **Models** | OpenAI (GPT-4, GPT-5) |
| **Maturity** | Active development |

**CLI Usage Patterns**:
```bash
# Basic usage (in git repo)
mentat

# With specific files for context
mentat file1.py file2.js
```

**Autospec Config Example**:
```yaml
claude_cmd: mentat
claude_args: []
```

**Strengths**:
- Project-wide context understanding
- Multi-file editing
- Git workflow support
- GitHub issue/PR integration

**Repo**: https://mentat.ai / https://github.com/AbanteAI/mentat

---

### Tier 4: IDE-First (Limited CLI Support)

These tools are primarily IDE extensions but have some CLI capabilities.

| Tool | CLI Status | Notes |
|------|------------|-------|
| **Cursor Agent** | `cursor-agent` exists but Background Agents CLI is feature request | Editor-first |
| **Windsurf/Codeium** | No standalone CLI | Use via `windsurf.vim` in Neovim |
| **Continue** | IDE extension primarily | Some CLI capabilities |
| **GitHub Copilot** | Editor/cloud first | `gh copilot` for suggestions |

---

## Comparison Matrix

### Tier 1: Automatable Agents (Have YOLO/Autonomous Mode)

| Agent | Prompt Arg | Autonomous Flag | MCP | Local Models | Open Source |
|-------|------------|-----------------|-----|--------------|-------------|
| **Claude Code** | `-p` | `--dangerously-skip-permissions` | Yes | No | No |
| **Cline** | positional | `-Y` (YOLO) or `-O` (one-shot) | No | Yes | Yes |
| **Gemini CLI** | `-p` | `--yolo` or `-y` | Yes | No | Yes |
| **Codex CLI** | `exec` subcommand | inherent | Yes | No | No |
| **OpenCode** | `run` subcommand | inherent | Yes | Yes | Yes |
| **Goose** | `run -t` | `GOOSE_MODE=auto --no-session` | Yes | Yes | Yes |

### Tier 2+: Needs Verification or Not Automatable

| Agent | Open Source | Local Models | Autonomous Mode | Notes |
|-------|-------------|--------------|-----------------|-------|
| ForgeCode | Yes | No | Unknown | Needs verification |
| Qwen CLI | Yes | Yes | Unknown | Needs verification |
| Grok CLI | Yes | No | Unknown | Needs verification |
| Rovo Dev | No | No | `--yolo` | Enterprise only |
| GPT Engineer | Yes | No | No | Greenfield generator, not agentic |
| Amazon Q | No | No | No | Interactive `q chat` only |
| Open Interpreter | Yes | Yes | Partial (`-y`) | May still prompt |

### Out of Scope (Not Agentic)

| Tool | Reason |
|------|--------|
| **Aider** | Pair programmer, not autonomous agent |
| **Mentat** | Limited non-interactive support |
| **Ollamacode** | Limited automation flags |

---

## Integration Recommendations

### Minimal Changes (Phase 1)

The current architecture already supports all Tier 1 agents via `custom_claude_cmd`:

```yaml
# Claude Code (default)
custom_claude_cmd: "claude -p {{PROMPT}} --dangerously-skip-permissions"

# Cline (YOLO mode)
custom_claude_cmd: "cline -Y {{PROMPT}}"

# Gemini CLI (YOLO mode)
custom_claude_cmd: "gemini -p {{PROMPT}} --yolo"

# Codex CLI (exec mode)
custom_claude_cmd: "codex exec {{PROMPT}}"

# OpenCode (run mode)
custom_claude_cmd: "opencode run {{PROMPT}}"

# Goose (autonomous headless mode)
custom_claude_cmd: "GOOSE_MODE=auto goose run --no-session -t {{PROMPT}}"
```

### Enhanced Support (Phase 2)

1. **Rename config fields** for agent-agnosticism:
   - `claude_cmd` → `agent_cmd`
   - `claude_args` → `agent_args`
   - `custom_claude_cmd` → `custom_agent_cmd`

2. **Add agent presets** in config schema:
   ```yaml
   agent_preset: aider  # or: claude, gemini, codex, cline, opencode, qwen, etc.
   ```

3. **Update `internal/commands/*.md`** templates:
   - Some agents may need different prompt formatting
   - Consider agent-specific command templates

### Agent-Specific Commands (Phase 3)

Create preset configurations in `internal/agent/`:

```go
// internal/agent/presets.go
var AgentPresets = map[string]AgentConfig{
    "claude": {
        Cmd:           "claude",
        PromptFlag:    "-p",
        AutonomousFlag: "--dangerously-skip-permissions",
    },
    "cline": {
        Cmd:           "cline",
        PromptFlag:    "",  // positional
        AutonomousFlag: "-Y",
    },
    "gemini": {
        Cmd:           "gemini",
        PromptFlag:    "-p",
        AutonomousFlag: "--yolo",
    },
    "codex": {
        Cmd:           "codex",
        Subcommand:    "exec",
        PromptFlag:    "",  // positional after subcommand
    },
    "opencode": {
        Cmd:           "opencode",
        Subcommand:    "run",
        PromptFlag:    "",  // positional after subcommand
    },
    "goose": {
        Cmd:           "goose",
        Subcommand:    "run",
        PromptFlag:    "-t",
        AutonomousFlag: "--no-session",
        AutonomousEnv:  map[string]string{"GOOSE_MODE": "auto"},
    },
}
```

---

## Next Steps

1. [ ] Test Tier 1 agents with autospec's `custom_claude_cmd`:
   - [ ] Claude Code (current default)
   - [ ] Cline (`-Y` YOLO mode)
   - [ ] Gemini CLI (`-p --yolo`)
   - [ ] Codex CLI (`exec` mode)
   - [ ] OpenCode (`run` subcommand)
   - [ ] Goose (`GOOSE_MODE=auto run --no-session -t`)
2. [ ] Document working configurations in `docs/agents.md`
3. [ ] Rename `claude_*` config fields to `agent_*` (with backward compat)
4. [ ] Add `--agent` flag or `agent_preset` config option
5. [ ] Implement `internal/agent/` package with interfaces and presets
6. [ ] Test local model support via OpenCode and Goose

---

## References

- Claude Code: https://github.com/anthropics/claude-code
- Aider: https://aider.chat / https://github.com/Aider-AI/aider
- Cline: https://cline.bot / https://docs.cline.bot/cline-cli/
- Gemini CLI: https://github.com/google-gemini/gemini-cli
- OpenAI Codex CLI: https://developers.openai.com/codex/cli/
- OpenCode: https://opencode.ai / https://github.com/sst/opencode
- ForgeCode: https://forgecode.dev / https://github.com/antinomyhq/forge
- Qwen CLI: https://github.com/QwenLM/qwen-code
- Grok CLI: https://github.com/superagent-ai/grok-cli
- Rovo Dev: https://support.atlassian.com/rovo/docs/rovo-dev-cli-commands/
- GPT Engineer: https://github.com/AntonOsika/gpt-engineer
- Goose: https://github.com/block/goose
- Amazon Q: https://docs.aws.amazon.com/amazonq/
- Open Interpreter: https://github.com/openinterpreter/open-interpreter
- Mentat: https://mentat.ai
- Artificial Analysis Coding Agents: https://artificialanalysis.ai/insights/coding-agents-comparison
- KDNuggets Top 5 Agentic CLI Tools: https://www.kdnuggets.com/top-5-agentic-coding-cli-tools
