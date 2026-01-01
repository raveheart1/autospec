# claude-clean (cclean)

*Last updated: 2025-12-20*

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

Set the output style in `~/.config/autospec/config.yml`:

```yaml
# Output formatting style for stream-json mode
output_style: default  # default | compact | minimal | plain | raw
```

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
2. **Config file** (`output_style:`) - Default when flag not set
3. **Built-in default** - `default` style

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
