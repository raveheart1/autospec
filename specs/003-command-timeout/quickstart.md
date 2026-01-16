# Quickstart: Command Execution Timeout

**Feature**: 003-command-timeout
**Date**: 2025-10-22
**Time to Complete**: 10 minutes

## Overview

This quickstart guide demonstrates how to configure and use the command timeout feature in autospec. The timeout feature automatically terminates long-running Claude commands to prevent indefinite hangs in automated workflows.

---

## Prerequisites

- `autospec` binary installed (version with timeout support)
- Basic familiarity with autospec commands
- Text editor for config file editing

---

## Quick Start (5 minutes)

### Step 1: Configure Timeout via Environment Variable

The fastest way to enable timeout is via environment variable:

```bash
# Set 5-minute timeout for all commands
export AUTOSPEC_TIMEOUT=300

# Run any autospec command
autospec workflow "Add user authentication"
```

**What happens**:
- All Claude commands will timeout after 5 minutes
- If timeout occurs, you'll see an error message with a hint to increase the timeout

---

### Step 2: Configure Timeout via Config File

For persistent configuration, add timeout to your config file:

**Create local config** (project-specific):
```bash
mkdir -p .autospec
cat > .autospec/config.json <<EOF
{
  "timeout": 300
}
EOF
```

**Or edit global config** (all projects):
```bash
mkdir -p ~/.autospec
cat > ~/.autospec/config.json <<EOF
{
  "timeout": 600
}
EOF
```

**Verify configuration**:
```bash
autospec config show
# Should display: "timeout": 300 (or your configured value)
```

---

### Step 3: Test Timeout Behavior

**Test successful completion** (command finishes before timeout):
```bash
# Set generous timeout
export AUTOSPEC_TIMEOUT=600

# Run quick command
autospec specify "simple feature"

# Expected: Command completes normally, no timeout
```

**Test timeout enforcement** (command exceeds timeout):
```bash
# Set very short timeout (for testing only!)
export AUTOSPEC_TIMEOUT=5

# Run command that typically takes longer
autospec workflow "complex feature"

# Expected after 5 seconds:
# Error: command timed out after 5s: claude /speckit.workflow ...
# Hint: increase timeout in config
# Exit code: 5
```

---

## Configuration Reference

### Valid Timeout Values

| Value | Meaning | Use Case |
|-------|---------|----------|
| 0 or missing | No timeout (default) | Development, interactive use |
| 30-60 | 30 seconds - 1 minute | Quick commands, testing |
| 300 | 5 minutes | Most workflow commands |
| 600 | 10 minutes | Complex planning/implementation |
| 1800 | 30 minutes | Very large features |
| 3600 | 1 hour (max) | Extremely complex tasks |

### Configuration Priority

Configuration sources are checked in this order (highest to lowest priority):

1. **Environment variable**: `AUTOSPEC_TIMEOUT`
2. **Local config**: `.autospec/config.json`
3. **Global config**: `~/.autospec/config.json`
4. **Default**: `0` (no timeout)

**Example** (multiple sources):
```bash
# Global config
~/.autospec/config.json: {"timeout": 600}

# Local config (overrides global)
.autospec/config.json: {"timeout": 300}

# Environment variable (overrides all)
export AUTOSPEC_TIMEOUT=120

# Result: timeout = 120 seconds
```

---

## Common Use Cases

### Use Case 1: CI/CD Pipeline

**Goal**: Prevent stuck builds in continuous integration.

**Recommended Timeout**: 5-10 minutes

**Configuration**:
```bash
# In CI script
export AUTOSPEC_TIMEOUT=600  # 10 minutes

# Run workflow
autospec workflow "feature description"

# Handle exit codes
if [ $? -eq 5 ]; then
    echo "Command timed out. Consider increasing timeout or optimizing workflow."
    exit 1
fi
```

---

### Use Case 2: Local Development

**Goal**: Allow long-running commands but catch truly stuck processes.

**Recommended Timeout**: 30 minutes or no timeout

**Configuration**:
```json
// ~/.autospec/config.json
{
  "timeout": 1800
}
```

**Or disable timeout entirely**:
```bash
unset AUTOSPEC_TIMEOUT
# Or set to 0
export AUTOSPEC_TIMEOUT=0
```

---

### Use Case 3: Automated Scripts

**Goal**: Timeout per command based on expected duration.

**Configuration**:
```bash
#!/bin/bash

# Quick specification (5 minutes)
AUTOSPEC_TIMEOUT=300 autospec specify "feature"

# Planning (10 minutes)
AUTOSPEC_TIMEOUT=600 autospec plan

# Long implementation (30 minutes)
AUTOSPEC_TIMEOUT=1800 autospec implement
```

---

## Troubleshooting

### Problem: Command times out unexpectedly

**Symptoms**:
- Error: "command timed out after Xs"
- Exit code: 5

**Solutions**:

1. **Increase timeout**:
   ```bash
   # Try doubling the timeout
   export AUTOSPEC_TIMEOUT=600
   ```

2. **Check command complexity**:
   - Large features may need more time
   - Complex planning phases take longer

3. **Disable timeout temporarily**:
   ```bash
   export AUTOSPEC_TIMEOUT=0
   autospec workflow "feature"
   ```

---

### Problem: Timeout not being respected

**Symptoms**:
- Command runs longer than configured timeout
- No timeout error occurs

**Diagnostics**:

1. **Verify configuration is loaded**:
   ```bash
   autospec config show | grep timeout
   ```

2. **Check environment variables**:
   ```bash
   echo $AUTOSPEC_TIMEOUT
   ```

3. **Verify autospec version**:
   ```bash
   autospec version
   # Should show version with timeout support
   ```

**Solutions**:
- Ensure config file is in correct location
- Check environment variable spelling: `AUTOSPEC_TIMEOUT` (all caps)
- Reload config or restart shell session

---

### Problem: Want different timeouts for different commands

**Solution**: Use environment variable per command:

```bash
# Specify: 5 minutes
AUTOSPEC_TIMEOUT=300 autospec specify "feature"

# Plan: 10 minutes
AUTOSPEC_TIMEOUT=600 autospec plan

# Implement: 30 minutes
AUTOSPEC_TIMEOUT=1800 autospec implement
```

**Or use shell script**:
```bash
#!/bin/bash
set -e

function run_with_timeout() {
    local timeout=$1
    shift
    AUTOSPEC_TIMEOUT=$timeout "$@"
}

run_with_timeout 300 autospec specify "feature"
run_with_timeout 600 autospec plan
run_with_timeout 1800 autospec implement
```

---

## Advanced Usage

### CLI Flag (If Implemented)

If `--timeout` flag is added:

```bash
# Override config with flag
autospec --timeout 300 workflow "feature"

# Short flag
autospec -t 300 plan
```

**Priority**: CLI flag > Environment > Config file

---

### Handling Timeout in Scripts

**Detect timeout errors**:
```bash
#!/bin/bash

autospec workflow "feature"
exit_code=$?

if [ $exit_code -eq 5 ]; then
    echo "Command timed out"
    echo "Consider increasing AUTOSPEC_TIMEOUT"
    exit 1
elif [ $exit_code -ne 0 ]; then
    echo "Command failed with exit code $exit_code"
    exit $exit_code
fi

echo "Command completed successfully"
```

**Retry with increased timeout**:
```bash
#!/bin/bash

timeout=300
max_retries=3

for i in $(seq 1 $max_retries); do
    echo "Attempt $i with timeout ${timeout}s"
    AUTOSPEC_TIMEOUT=$timeout autospec workflow "feature"
    exit_code=$?

    if [ $exit_code -eq 5 ]; then
        echo "Timed out, increasing timeout and retrying..."
        timeout=$((timeout * 2))
    elif [ $exit_code -eq 0 ]; then
        echo "Success!"
        exit 0
    else
        echo "Failed with exit code $exit_code"
        exit $exit_code
    fi
done

echo "All retries exhausted"
exit 1
```

---

## Best Practices

### 1. Start with Conservative Timeout

**Recommendation**: Start with 10 minutes (600s) and adjust based on experience.

```bash
export AUTOSPEC_TIMEOUT=600
```

### 2. Use Different Timeouts per Environment

**Development** (no timeout or generous timeout):
```bash
export AUTOSPEC_TIMEOUT=0
```

**CI/CD** (strict timeout):
```bash
export AUTOSPEC_TIMEOUT=300
```

### 3. Monitor Actual Command Duration

**Track how long commands take**:
```bash
time autospec workflow "feature"
# Use this data to set appropriate timeouts
```

### 4. Document Timeout Requirements

**In project README**:
```markdown
## Configuration

This project's workflows typically complete in:
- Specify: 2-3 minutes
- Plan: 5-8 minutes
- Tasks: 2-3 minutes
- Implement: 15-20 minutes

Recommended timeout: 600 seconds (10 minutes) for individual commands
```

---

## What's Next?

After completing this quickstart:

1. **Review full documentation**: See `CLAUDE.md` for detailed timeout behavior
2. **Run tests**: Verify timeout enforcement works correctly
3. **Adjust timeouts**: Tune timeout values based on your workflow needs
4. **Monitor CI/CD**: Check for timeout-related build failures

---

## Summary

**Key Commands**:
```bash
# Set timeout via environment
export AUTOSPEC_TIMEOUT=300

# Set timeout via config file
echo '{"timeout": 300}' > .autospec/config.json

# Verify configuration
autospec config show

# Test timeout
AUTOSPEC_TIMEOUT=5 autospec workflow "test"
```

**Key Concepts**:
- Timeout in seconds (1-3600)
- 0 or missing = no timeout (default)
- Exit code 5 = timeout occurred
- Environment variable overrides config files

**Next Steps**:
- Configure timeout for your environment
- Test with short timeout to verify behavior
- Set appropriate production timeouts
- Add timeout handling to CI/CD scripts
