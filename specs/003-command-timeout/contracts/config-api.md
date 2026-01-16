# Config API Contract: Timeout Configuration

**Package**: `internal/config`
**Version**: 1.0.0
**Feature**: 003-command-timeout

## Overview

This contract defines the configuration API for timeout functionality. The timeout configuration integrates with the existing koanf-based configuration system and follows the established configuration hierarchy.

---

## Configuration Structure

### Configuration Struct

**Package**: `internal/config`
**Type**: `Configuration`

```go
type Configuration struct {
    // Existing fields...
    ClaudeCmd       string   `koanf:"claude_cmd" validate:"required"`
    ClaudeArgs      []string `koanf:"claude_args"`
    UseAPIKey       bool     `koanf:"use_api_key"`
    CustomClaudeCmd string   `koanf:"custom_claude_cmd"`
    SpecifyCmd      string   `koanf:"specify_cmd" validate:"required"`
    MaxRetries      int      `koanf:"max_retries" validate:"min=1,max=10"`
    SpecsDir        string   `koanf:"specs_dir" validate:"required"`
    StateDir        string   `koanf:"state_dir" validate:"required"`
    SkipPreflight   bool     `koanf:"skip_preflight"`

    // NEW: Timeout configuration
    Timeout         int      `koanf:"timeout" validate:"omitempty,min=1,max=3600"`
}
```

**Field Details**:

| Field | Type | Required | Validation | Default | Description |
|-------|------|----------|------------|---------|-------------|
| Timeout | int | No | 1 ≤ x ≤ 3600 (if present) | 0 | Command timeout in seconds. 0 or missing = no timeout |

---

## Configuration Sources

Configuration is loaded from multiple sources with the following priority order:

1. **Environment Variables** (Highest priority)
   - Variable: `AUTOSPEC_TIMEOUT`
   - Format: Integer seconds
   - Example: `AUTOSPEC_TIMEOUT=300`

2. **Local Configuration File**
   - Path: `./.autospec/config.json`
   - Format: JSON
   - Example:
     ```json
     {
       "timeout": 300
     }
     ```

3. **Global Configuration File**
   - Path: `~/.autospec/config.json`
   - Format: JSON
   - Example:
     ```json
     {
       "timeout": 600
     }
     ```

4. **Default Values** (Lowest priority)
   - Value: `0` (no timeout)
   - Defined in: `internal/config/defaults.go`

---

## API Functions

### Load Configuration

**Function**: `Load(localConfigPath string) (*Configuration, error)`
**Package**: `internal/config`

**Purpose**: Load configuration from all sources and return validated Configuration struct.

**Parameters**:
- `localConfigPath`: Path to local config file (typically `.autospec/config.json`)

**Returns**:
- `*Configuration`: Fully loaded and validated configuration
- `error`: Validation error if timeout value is invalid

**Behavior**:
1. Apply default values (timeout = 0)
2. Load global config if exists
3. Load local config if exists
4. Override with environment variables
5. Validate all fields including timeout
6. Return error if validation fails

**Example Usage**:
```go
cfg, err := config.Load(".autospec/config.json")
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// Access timeout value
timeout := cfg.Timeout  // 0 = no timeout, >0 = timeout in seconds
```

**Error Cases**:

| Error | Cause | Example |
|-------|-------|---------|
| Validation error | Timeout < 1 | `AUTOSPEC_TIMEOUT=-5` |
| Validation error | Timeout > 3600 | `AUTOSPEC_TIMEOUT=7200` |
| Parse error | Non-numeric value | `AUTOSPEC_TIMEOUT=invalid` |

---

## Validation Rules

### Timeout Field Validation

**Validator Tag**: `validate:"omitempty,min=1,max=3600"`

**Rules**:
1. **omitempty**: Field is optional; 0 or missing is valid (means no timeout)
2. **min=1**: If present, must be at least 1 second
3. **max=3600**: If present, must not exceed 3600 seconds (1 hour)

**Valid Values**:
- `0`: No timeout (default, backward compatible)
- `1` to `3600`: Timeout in seconds (1 second to 1 hour)

**Invalid Values**:
- Negative numbers: `-1`, `-100`
- Zero (when explicitly validated): Not applicable (zero is valid = no timeout)
- Too large: `3601`, `86400` (24 hours)
- Non-numeric: `"5m"`, `"invalid"`

**Validation Enforcement**:
- Performed by `validator.v10` package
- Executed in `Load()` function after unmarshaling
- Returns descriptive error on validation failure

---

## Configuration Hierarchy Example

**Scenario**: Multiple configuration sources present

**Global Config** (`~/.autospec/config.json`):
```json
{
  "timeout": 600,
  "max_retries": 3
}
```

**Local Config** (`.autospec/config.json`):
```json
{
  "timeout": 300
}
```

**Environment Variable**:
```bash
export AUTOSPEC_TIMEOUT=120
```

**Result**:
```go
cfg.Timeout = 120  // Environment variable wins (highest priority)
cfg.MaxRetries = 3 // From global config (not overridden)
```

---

## Backward Compatibility

### Missing Timeout Configuration

**Scenario**: User has existing config without timeout field

**Example** (`config.json`):
```json
{
  "claude_cmd": "claude",
  "max_retries": 3
}
```

**Behavior**:
- `cfg.Timeout` defaults to `0`
- No timeout enforcement
- Existing behavior preserved (infinite wait)
- No migration required

### Explicit Zero Timeout

**Scenario**: User explicitly sets timeout to 0

**Example**:
```json
{
  "timeout": 0
}
```

**Behavior**:
- Same as missing timeout (no enforcement)
- `omitempty` validator skips validation for 0
- Backward compatible

---

## CLI Flag (Optional Enhancement)

### Global Timeout Flag

**Flag**: `--timeout`
**Type**: `int`
**Scope**: Global flag (applies to all commands)

**Example**:
```bash
autospec --timeout 300 workflow "feature description"
autospec -t 300 plan
```

**Priority**:
- CLI flag > Environment variable > Local config > Global config > Default

**Implementation**:
```go
// In internal/cli/root.go
rootCmd.PersistentFlags().IntP("timeout", "t", 0, "Command timeout in seconds (0 = no timeout)")
```

**Note**: This is optional and not required for MVP. Config file and environment variable are sufficient.

---

## Testing Contract

### Test Cases

1. **Load with valid timeout**
   - Input: `{"timeout": 300}`
   - Expected: `cfg.Timeout = 300`

2. **Load with missing timeout**
   - Input: `{}`
   - Expected: `cfg.Timeout = 0`

3. **Load with timeout = 0**
   - Input: `{"timeout": 0}`
   - Expected: `cfg.Timeout = 0` (no error)

4. **Load with invalid timeout (negative)**
   - Input: `{"timeout": -1}`
   - Expected: Validation error

5. **Load with invalid timeout (too large)**
   - Input: `{"timeout": 7200}`
   - Expected: Validation error

6. **Environment variable override**
   - Env: `AUTOSPEC_TIMEOUT=120`
   - Config: `{"timeout": 300}`
   - Expected: `cfg.Timeout = 120`

7. **Parse error (non-numeric)**
   - Env: `AUTOSPEC_TIMEOUT=invalid`
   - Expected: Parse error or validation error

---

## Error Messages

### Validation Error

**Format**:
```
config validation failed: Key: 'Configuration.Timeout' Error:Field validation for 'Timeout' failed on the 'min' tag
```

**User-Friendly Message**:
```
Error: Invalid timeout value. Timeout must be between 1 and 3600 seconds (1 hour).
```

### Parse Error

**Format**:
```
failed to unmarshal config: json: cannot unmarshal string into Go struct field Configuration.timeout of type int
```

**User-Friendly Message**:
```
Error: Invalid timeout format. Timeout must be an integer (seconds).
```

---

## Summary

- **Field**: `Timeout int` in `Configuration` struct
- **Config Key**: `timeout`
- **Environment Variable**: `AUTOSPEC_TIMEOUT`
- **Validation**: `1 ≤ timeout ≤ 3600` (if present), `0` = no timeout
- **Default**: `0` (no timeout, backward compatible)
- **Priority**: Env > Local > Global > Default
