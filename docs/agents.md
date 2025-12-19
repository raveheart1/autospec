# CLI Agent Configuration

autospec supports multiple CLI-based AI coding agents through a unified agent abstraction layer. This allows you to use your preferred agent while maintaining compatibility with the same workflow commands.

## Supported Agents

### Tier 1 Agents (Built-in)

| Agent | Binary | Description |
|-------|--------|-------------|
| `claude` | `claude` | Anthropic's Claude Code CLI (default) |
| `cline` | `cline` | Cline VSCode extension CLI |
| `gemini` | `gemini` | Google Gemini CLI |
| `codex` | `codex` | OpenAI Codex CLI |
| `opencode` | `opencode` | OpenCode CLI |
| `goose` | `goose` | Goose AI CLI |

All built-in agents support headless/automated execution suitable for CI/CD pipelines.

### Custom Agents

You can configure any CLI tool as an agent using a command template with `{{PROMPT}}` placeholder.

## Configuration

### Using a Preset Agent

Set the `agent_preset` field in your configuration file:

```yaml
# .autospec/config.yml
agent_preset: claude
```

Or in user-level config:

```yaml
# ~/.config/autospec/config.yml
agent_preset: gemini
```

### Using a Custom Agent Command

For agents not built-in, or for custom configurations:

```yaml
# .autospec/config.yml
custom_agent_cmd: "my-agent run --prompt {{PROMPT}} --mode headless"
```

The `{{PROMPT}}` placeholder is replaced with the actual prompt at execution time. The placeholder can appear anywhere in the command template.

### CLI Flag Override

Override the configured agent for a single command execution:

```bash
# Use gemini for this run only
autospec run -a "Add user auth" --agent gemini

# Use codex for planning only
autospec plan --agent codex

# Use cline for implementation
autospec implement --agent cline
```

Available for all workflow commands: `run`, `prep`, `specify`, `plan`, `tasks`, `implement`.

## Configuration Priority

When determining which agent to use, autospec follows this priority order:

1. **CLI flag** (`--agent`): Highest priority, single-command override
2. **custom_agent_cmd**: Project or user-level custom command template
3. **agent_preset**: Project or user-level preset name
4. **Legacy fields** (deprecated): `custom_claude_cmd`, `claude_cmd`
5. **Default**: Falls back to `claude` agent

## Environment Configuration

Override agent settings via environment variables:

```bash
# Set agent preset
export AUTOSPEC_AGENT_PRESET=gemini

# Set custom agent command
export AUTOSPEC_CUSTOM_AGENT_CMD="my-agent --prompt {{PROMPT}}"
```

Environment variables take precedence over config file values.

## Agent Requirements

Each agent has specific requirements:

| Agent | Binary in PATH | Environment Variables |
|-------|----------------|----------------------|
| `claude` | `claude` | - |
| `cline` | `cline` | - |
| `gemini` | `gemini` | `GOOGLE_API_KEY` |
| `codex` | `codex` | `OPENAI_API_KEY` |
| `opencode` | `opencode` | - |
| `goose` | `goose` | - |

Use `autospec doctor` to verify agent availability and configuration.

## Checking Agent Status

The `autospec doctor` command shows the status of all registered agents:

```bash
$ autospec doctor

Dependencies:
  Git: installed
  Claude CLI: installed

CLI Agents:
  claude: installed (v1.0.5)
  cline: not found in PATH
  codex: missing OPENAI_API_KEY environment variable
  gemini: installed (v0.8.2)
  goose: not found in PATH
  opencode: installed (v2.1.0)
```

## Migration from Legacy Configuration

If you're using the older `claude_cmd` or `custom_claude_cmd` fields, consider migrating to the new agent configuration:

### Before (deprecated)

```yaml
# Old configuration
claude_cmd: claude
claude_args:
  - --model
  - opus
custom_claude_cmd: "claude -p {{PROMPT}} | tee output.log"
```

### After (recommended)

```yaml
# New configuration
agent_preset: claude

# Or for custom command
custom_agent_cmd: "claude -p {{PROMPT}} | tee output.log"
```

### Deprecation Warnings

When using legacy fields, autospec displays deprecation warnings on stderr:

```
Deprecation warning: 'claude_cmd' is deprecated, use 'agent_preset: claude' instead
Deprecation warning: 'custom_claude_cmd' is deprecated, use 'custom_agent_cmd' instead
```

The legacy fields continue to work but will be removed in a future version.

## Custom Agent Examples

### Using a Custom Model with Claude

```yaml
custom_agent_cmd: "claude --model claude-3-opus {{PROMPT}}"
```

### Piping Output Through a Filter

```yaml
custom_agent_cmd: "claude -p {{PROMPT}} | grep -v DEBUG"
```

### Using SSH to Run on Remote Machine

```yaml
custom_agent_cmd: "ssh build-server 'claude -p {{PROMPT}}'"
```

### Using Docker Container

```yaml
custom_agent_cmd: "docker run --rm ai-agent run {{PROMPT}}"
```

## Agent Capabilities

All agents expose their capabilities through the agent abstraction:

| Capability | Description |
|------------|-------------|
| Automatable | Supports headless/non-interactive execution |
| Interactive | Supports interactive prompts (not used by autospec) |
| Streaming | Supports real-time output streaming |

Currently, autospec requires automatable agents for all workflow commands.

## Troubleshooting

### Agent Not Found

If `autospec doctor` shows an agent as "not found in PATH":

1. Verify the agent binary is installed
2. Ensure the binary is in your system PATH
3. Try running the agent directly: `which claude` or `claude --version`

### Missing Environment Variables

Some agents require API keys or configuration:

```bash
# For Gemini
export GOOGLE_API_KEY=your-api-key

# For Codex
export OPENAI_API_KEY=your-api-key
```

### Custom Agent Template Issues

If your custom agent command isn't working:

1. Verify `{{PROMPT}}` placeholder is present in the template
2. Test the command manually with a simple prompt
3. Check shell quoting and escaping

```bash
# Test custom command manually
my-agent run --prompt "test prompt"
```

### Agent Validation Failed

If agent validation fails, check:

1. Binary exists and is executable
2. Required environment variables are set
3. Agent can run with `--version` or similar flag
