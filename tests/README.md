# SpecKit Validation Tests

This directory contains tests for the SpecKit validation hooks and scripts using the bats-core testing framework.

## Installation

### bats-core Installation

**Option 1: Using npm (recommended for development)**

```bash
npm install -g bats
```

**Option 2: Using package manager**

**Ubuntu/Debian:**

```bash
sudo apt-get install bats
```

**macOS:**

```bash
brew install bats-core
```

**Arch Linux:**

```bash
sudo pacman -S bats
```

**Manual Installation:**

```bash
git clone https://github.com/bats-core/bats-core.git
cd bats-core
sudo ./install.sh /usr/local
```

### Verify Installation

```bash
bats --version
# Should output: Bats 1.10.0 or higher
```

## Running Tests

### Run all tests

```bash
bats tests/
```

### Run specific test suite

```bash
bats tests/lib/validation-lib.bats
bats tests/scripts/workflow-validate.bats
bats tests/hooks/stop-speckit-implement.bats
```

### Run with verbose output

```bash
bats -t tests/
```

## Test Structure

```
tests/
├── hooks/              # Stop hook script tests
│   ├── stop-speckit-specify.bats
│   ├── stop-speckit-plan.bats
│   ├── stop-speckit-tasks.bats
│   ├── stop-speckit-implement.bats
│   └── stop-speckit-clarify.bats
├── scripts/            # Workflow script tests
│   ├── workflow-validate.bats
│   └── implement-validate.bats
├── lib/                # Library function tests
│   └── validation-lib.bats
├── fixtures/           # Test data
│   ├── mock-spec.md
│   ├── mock-tasks-complete.md
│   ├── mock-tasks-incomplete.md
│   └── mock-hook-payloads.json
└── mocks/              # Mock external commands
    └── mock-claude.sh
```

## Writing Tests

### Example Test

```bash
#!/usr/bin/env bats

# Load test helpers
load test_helper

@test "validate_file_exists returns 0 when file exists" {
  # Setup
  touch "$BATS_TMPDIR/test-file.md"

  # Execute
  run validate_file_exists "test-file.md" "$BATS_TMPDIR"

  # Assert
  [ "$status" -eq 0 ]
}
```

## Test Fixtures

Mock files are available in `fixtures/` directory for testing validation logic with realistic data.

## Continuous Integration

Tests can be run in CI/CD pipelines:

```yaml
# .github/workflows/test.yml
- name: Run validation tests
  run: |
      npm install -g bats
      bats tests/
```
