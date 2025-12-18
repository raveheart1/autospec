# Config Set Command

Add CLI commands to edit/toggle configuration values without manually editing YAML files.

## Problem

Currently, to change configuration values users must:
1. Manually edit `~/.config/autospec/config.yml` (user-level)
2. Manually edit `.autospec/config.yml` (project-level)
3. Use environment variables for temporary overrides

There's no CLI command to persistently set/toggle configuration values.

## Proposed Solution

Add `autospec config set` subcommand:

```bash
# Set a value (user-level by default)
autospec config set notifications.enabled true
autospec config set max_retries 5

# Set at project level
autospec config set notifications.enabled true --project

# Explicit user level
autospec config set timeout 10m --user
```

### Scope Options

| Flag | Target File |
|------|-------------|
| `--user` | `~/.config/autospec/config.yml` |
| `--project` | `.autospec/config.yml` |
| (default) | User config |

### Type Inference

The command should infer types from the value:
- `true`/`false` → boolean
- Numbers → integer
- Duration strings (`5m`, `1h`) → string (validated as duration)
- Everything else → string

### Supported Keys

Common configuration keys to support:
- `notifications.enabled` (bool)
- `notifications.type` (string: "sound", "visual", "both")
- `notifications.on_command_complete` (bool)
- `notifications.on_error` (bool)
- `notifications.on_long_running` (bool)
- `notifications.long_running_threshold` (duration)
- `max_retries` (int)
- `timeout` (duration)
- `skip_preflight` (bool)
- `skip_confirmations` (bool)
- `claude_cmd` (string)
- `specs_dir` (string)

## Implementation Notes

### File Handling
1. Read existing YAML (or create new if missing)
2. Parse dotted key path (e.g., `notifications.enabled` → nested map)
3. Set value with correct type
4. Write back preserving comments where possible (use `gopkg.in/yaml.v3` node API)

### Validation
- Validate key exists in schema
- Validate value type matches expected type
- For enums (like `notifications.type`), validate against allowed values

### Edge Cases
- Creating nested structure when parent keys don't exist
- Handling invalid key paths gracefully
- Config file doesn't exist yet → create it
- Final task must be to add this change ONE line to CHANGELOG.md

## Alternative: Toggle Command

For boolean values, a dedicated toggle might be useful:

```bash
autospec config toggle notifications.enabled
autospec config toggle skip_preflight --project
```

## Testing

- Unit tests for key path parsing
- Unit tests for type inference
- Integration tests for file read/write roundtrip
- Tests for creating config when missing
- Tests for nested key creation

## Priority

Medium - Nice to have for UX, but env vars provide a workaround.
