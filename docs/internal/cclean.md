# claude-clean (cclean)

*Last updated: 2026-01-02*

[claude-clean](https://github.com/ariel-frischer/claude-clean) transforms Claude Code's streaming JSON output into readable terminal output.

## Installation

claude-clean is bundled as a dependency of autospec. For standalone use:

```bash
go install github.com/ariel-frischer/claude-clean@latest
```

## CLI Usage

```bash
# Parse a JSONL conversation file
cclean output.jsonl

# Plain text output (best for piping/analysis)
cclean -s plain output.jsonl

# Available styles
cclean -s default output.jsonl   # Box-drawing characters (default)
cclean -s compact output.jsonl   # Single-line summaries
cclean -s minimal output.jsonl   # No box-drawing
cclean -s plain output.jsonl     # No colors

# With line numbers
cclean -n output.jsonl

# Verbose output (includes usage stats, tool IDs)
cclean -V output.jsonl
```

## Go Library Usage

```go
import (
    "github.com/ariel-frischer/claude-clean/parser"
    "github.com/ariel-frischer/claude-clean/display"
)

// Parse a JSONL line
var msg parser.StreamMessage
json.Unmarshal([]byte(line), &msg)

// Strip system reminders from text
clean := parser.StripSystemReminders(msg.Message.Content[0].Text)

// Display with styling
cfg := &display.Config{
    Style:       display.StyleDefault,
    Verbose:     false,
    LineNumbers: true,
}
display.DisplayMessage(&msg, 1, cfg)
```

## Display Styles

| Style | Constant | Description |
|-------|----------|-------------|
| default | `display.StyleDefault` | Box-drawing characters, full formatting |
| compact | `display.StyleCompact` | Single-line summaries |
| minimal | `display.StyleMinimal` | No box-drawing |
| plain | `display.StylePlain` | No colors, suitable for piping |

## Key Types

| Type | Package | Description |
|------|---------|-------------|
| `StreamMessage` | `parser` | Top-level message wrapper |
| `ContentBlock` | `parser` | Text, tool_use, or tool_result |
| `Config` | `display` | Output configuration |

## Native Integration with autospec

As of v0.2.0, autospec has native cclean integration. When your agent uses `--output-format stream-json` with headless mode (`-p`), autospec automatically formats the output using cclean.

### Configuration

The `cclean` section in your config file provides fine-grained control over output formatting.

**Config File Examples**:

Project-level (`.autospec/config.yml`):
```yaml
cclean:
  verbose: true           # Enable verbose output with usage stats and tool IDs
  line_numbers: true      # Show line numbers in formatted output
  style: compact          # Output style: default | compact | minimal | plain
```

User-level (`~/.config/autospec/config.yml`):
```yaml
cclean:
  verbose: false          # Quiet mode (default)
  line_numbers: false     # No line numbers (default)
  style: default          # Full formatting with box-drawing (default)
```

**Environment Variables**:

Override config file settings using environment variables:

```bash
# Enable verbose output
export AUTOSPEC_CCLEAN_VERBOSE=true

# Enable line numbers
export AUTOSPEC_CCLEAN_LINE_NUMBERS=true

# Set output style
export AUTOSPEC_CCLEAN_STYLE=minimal
```

### Cclean Configuration Options (v0.2.0)

| Option | Type | Default | Flag Equivalent | Description |
|--------|------|---------|-----------------|-------------|
| `cclean.verbose` | bool | `false` | `-V` | Enable verbose output with usage statistics and tool IDs |
| `cclean.line_numbers` | bool | `false` | `-n` | Show line numbers in formatted output |
| `cclean.style` | string | `default` | `-s` | Output formatting style |

**Allowed Values for `cclean.style`**:
- `default` - Full output with colored boxes and borders
- `compact` - Single-line summaries for each message
- `minimal` - No box-drawing characters
- `plain` - No colors (suitable for piping to files)

### CLI Flag

Override the config with `--output-style` on any workflow command:

```bash
# Use compact style for this run
autospec implement --output-style compact

# Raw output (bypass formatting, show raw JSONL)
autospec run -a "feature" --output-style raw
```

### Priority Order

1. **CLI flag** (`--output-style`) - Highest priority
2. **Environment variables** (`AUTOSPEC_CCLEAN_*`) - Override config files
3. **Project config** (`.autospec/config.yml`) - Project-specific settings
4. **User config** (`~/.config/autospec/config.yml`) - User defaults
5. **Built-in default** - `verbose=false`, `line_numbers=false`, `style=default`

### Default Values and Fallback Behavior

When no configuration is provided, the following defaults are used:

| Option | Default Value |
|--------|---------------|
| `verbose` | `false` |
| `line_numbers` | `false` |
| `style` | `default` |

**Fallback for Invalid Values**:
- Invalid `style` value (e.g., "fancy"): Logs a warning and falls back to `default`
- Non-boolean value for `verbose`/`line_numbers`: Logs a warning and falls back to `false`

### Output Styles

| Style | Description | Use Case |
|-------|-------------|----------|
| `default` | Box-drawing characters, colors | Interactive terminal |
| `compact` | Single-line summaries | Quick overview |
| `minimal` | Reduced visual output | Less clutter |
| `plain` | No colors | Piping, file output |
| `raw` | Bypasses formatting entirely | Debugging, log files |

### Automatic Detection

Formatting is applied only when both conditions are met:
- Agent uses `--output-format stream-json`
- Agent uses `-p` (headless/print mode)

If your agent doesn't use stream-json, output passes through unchanged.

### Legacy: External Post-Processor

For older setups or custom pipelines, you can still pipe through cclean externally:

```yaml
custom_agent:
  command: "claude"
  args:
    - "-p"
    - "--output-format"
    - "stream-json"
    - "{{PROMPT}}"
  post_processor: "cclean"
```

This approach is deprecated in favor of native integration.

## References

- [GitHub Repository](https://github.com/ariel-frischer/claude-clean)
- [Claude Settings & Sandboxing](claude-settings.md)
