# Auto Claude SpecKit

Automated validation and workflow scripts for Claude Code's SpecKit feature development system.

## What is This?

Auto Claude SpecKit provides bash scripts and Claude Code hooks that automate the validation of SpecKit workflows. It ensures your feature specifications, plans, and tasks are complete before allowing Claude to stop, and automatically retries commands when artifacts are missing.

### Key Features

- **Automated Workflow Validation**: Runs `/speckit` commands with automatic retry when outputs are missing
- **Hook-Based Enforcement**: Prevents Claude from stopping until required artifacts exist
- **Phase Completion Detection**: Validates implementation progress in `tasks.md`
- **Continuation Prompts**: Automatically generates prompts to resume incomplete work
- **Performance Optimized**: Sub-second validation times (0.22s average)
- **Comprehensive Testing**: 60+ tests with bats-core framework

## Quick Start

### Installation

#### Prerequisites

**Required:**
- Claude Code CLI (see https://www.claude.com/product/claude-code)
- SpecKit CLI (`uv tool install specify-cli --from git+https://github.com/github/spec-kit.git`)
- Git
- Go 1.21+ (for building from source)

**Optional:**
- jq (JSON processor)
- bats-core (for running tests)

See [PREREQUISITES.md](PREREQUISITES.md) for detailed installation instructions and platform-specific requirements.

#### Auto Claude SpecKit Setup

##### Option 1: Install Pre-Built Binary (Recommended)

Download the latest release for your platform:

```bash
# Linux (amd64)
curl -L https://github.com/anthropics/auto-claude-speckit/releases/latest/download/autospec-linux-amd64 -o autospec
chmod +x autospec
sudo mv autospec /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/anthropics/auto-claude-speckit/releases/latest/download/autospec-darwin-amd64 -o autospec
chmod +x autospec
sudo mv autospec /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/anthropics/auto-claude-speckit/releases/latest/download/autospec-darwin-arm64 -o autospec
chmod +x autospec
sudo mv autospec /usr/local/bin/

# Verify installation
autospec version
```

**Binary size**: ~2.5 MB per platform
**Startup time**: <50ms
**Platforms**: Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64)

##### Option 2: Install from Source with Go

If you have Go 1.21+ installed:

```bash
go install github.com/anthropics/auto-claude-speckit/cmd/autospec@latest

# Verify installation
autospec version
```

##### Option 3: Build from Source

1. Clone this repository:
```bash
git clone https://github.com/anthropics/auto-claude-speckit.git
cd auto-claude-speckit
```

2. Build the binary:
```bash
# Build for your current platform
go build -o autospec ./cmd/autospec

# Or build for all platforms
./scripts/build-all.sh

# Install locally
sudo mv autospec /usr/local/bin/

# Verify installation
autospec version
```

##### Legacy Bash Scripts (Deprecated)

The original bash scripts are still available in the `legacy/` directory but are deprecated in favor of the Go binary. They require additional dependencies:

```bash
# Install bats-core for testing (optional)
npm install -g bats

# Ensure dependencies are installed
command -v jq git grep sed >/dev/null && echo "All dependencies found"
```

### Initial Setup

Before using the SpecKit commands in Claude Code, you must initialize your project with the Specify templates:

```bash
# Non-interactive initialization (recommended for automation)
specify init . --ai claude --force

# Interactive initialization (choose AI assistant)
specify init .

# Check that all required tools are installed
specify check
```

**What this does:**
- Downloads the latest SpecKit templates from GitHub
- Sets up `.claude/commands/speckit.*.md` slash commands
- Configures project structure for feature development
- Initializes git repository (if not already present)

**Options:**
- `--ai claude` - Skip interactive AI selection, use Claude
- `--force` - Skip confirmation if directory not empty
- `--no-git` - Skip git repository initialization
- `--ignore-agent-tools` - Skip checks for AI agent tools like Claude Code

After initialization, you can use the `/speckit.*` commands in Claude Code.

### Basic Usage

#### 1. Automated Workflow with Validation

Run a complete SpecKit workflow with automatic validation and retry:

```bash
./scripts/speckit-workflow-validate.sh my-feature
```

This will:
- Create `specs/my-feature/` directory
- Run `/speckit.specify` and validate `spec.md` exists
- Run `/speckit.plan` and validate `plan.md` exists
- Run `/speckit.tasks` and validate `tasks.md` exists
- Automatically retry up to 3 times if any file is missing

**Options:**
- `--max-retries N`: Set maximum retry attempts (default: 3)
- `--dry-run`: Show what would be executed without running commands
- `--spec-name NAME`: Override automatic spec name detection

#### 2. Implementation Phase Validation

Check if implementation phases are complete and get continuation prompts:

```bash
./scripts/speckit-implement-validate.sh my-feature
```

**Example output (incomplete):**
```
Implementation Status: INCOMPLETE

Total tasks: 45
Completed: 23
Remaining: 22

Incomplete phases:
- Phase 3: User Story 1 (8 tasks remaining)
- Phase 4: User Story 2 (14 tasks remaining)

Next steps:
Continue implementing Phase 3: User Story 1
Focus on completing these tasks:
- [ ] T015 Create user authentication module
- [ ] T016 Add login validation
...
```

**Example output (complete):**
```
Implementation Status: COMPLETE

All 45 tasks completed across 6 phases.
Ready for final review and testing.
```

**Options:**
- `--json`: Output results in JSON format for programmatic use
- `--spec-name NAME`: Validate specific spec (default: auto-detect from branch)

#### 3. Hook-Based Automatic Validation

Enable hooks to prevent Claude from stopping until artifacts are complete:

```bash
# 1. Copy settings template
cp .claude/spec-workflow-settings.json .claude/my-workflow-settings.json

# 2. Edit to add desired hook (e.g., stop-speckit-specify.sh)
# Replace {{HOOK_SCRIPT}} with: /full/path/to/scripts/hooks/stop-speckit-specify.sh

# 3. Launch Claude with isolated settings
claude --settings .claude/my-workflow-settings.json
```

**Available hooks:**
- `stop-speckit-specify.sh`: Ensures `spec.md` exists before stopping
- `stop-speckit-plan.sh`: Ensures `plan.md` exists
- `stop-speckit-tasks.sh`: Ensures `tasks.md` exists
- `stop-speckit-implement.sh`: Ensures all implementation phases complete
- `stop-speckit-clarify.sh`: Ensures clarifications are captured

Each hook automatically retries up to 3 times before blocking.

## Architecture

### Core Components

1. **Validation Library** (`scripts/lib/speckit-validation-lib.sh`)
   - File existence validation
   - Retry state management
   - Task counting and phase parsing
   - Continuation prompt generation
   - Exit code conventions (0=success, 1=failed, 2=exhausted, 3=invalid, 4=missing deps)

2. **Workflow Script** (`scripts/speckit-workflow-validate.sh`)
   - Executes complete SpecKit workflow
   - Validates each command's output
   - Implements automatic retry logic
   - Supports dry-run mode

3. **Implementation Validator** (`scripts/speckit-implement-validate.sh`)
   - Parses `tasks.md` markdown structure
   - Counts unchecked tasks per phase
   - Detects phase completion
   - Generates context-aware continuation prompts

4. **Hook Scripts** (`scripts/hooks/stop-speckit-*.sh`)
   - Integrates with Claude Code's Stop hooks
   - Blocks premature stopping
   - Manages retry state
   - Provides helpful error messages

### Directory Structure

```
auto-claude-speckit/
├── scripts/
│   ├── lib/
│   │   └── speckit-validation-lib.sh    # Core validation functions
│   ├── hooks/
│   │   ├── stop-speckit-specify.sh      # Specification hook
│   │   ├── stop-speckit-plan.sh         # Planning hook
│   │   ├── stop-speckit-tasks.sh        # Task generation hook
│   │   ├── stop-speckit-implement.sh    # Implementation hook
│   │   └── stop-speckit-clarify.sh      # Clarification hook
│   ├── speckit-workflow-validate.sh     # Automated workflow runner
│   └── speckit-implement-validate.sh    # Implementation validator
├── tests/
│   ├── lib/
│   │   └── validation-lib.bats          # Library unit tests
│   ├── scripts/
│   │   ├── workflow-validate.bats       # Workflow tests
│   │   └── implement-validate.bats      # Implementation tests
│   ├── hooks/
│   │   └── stop-speckit-*.bats          # Hook tests
│   ├── fixtures/                        # Test fixtures
│   ├── mocks/                           # Mock scripts
│   ├── test_helper.bash                 # Test utilities
│   ├── quickstart-validation.bats       # Quickstart examples
│   ├── integration.bats                 # End-to-end tests
│   ├── run-all-tests.sh                 # Test runner
│   └── README.md                        # Testing guide
├── .claude/
│   └── spec-workflow-settings.json      # Settings template
└── README.md
```

## Testing

### Running Tests

```bash
# Run all tests
./tests/run-all-tests.sh

# Run specific test suite
bats tests/lib/validation-lib.bats
bats tests/scripts/workflow-validate.bats
bats tests/hooks/stop-speckit-implement.bats

# Run with verbose output
bats -t tests/integration.bats
```

### Test Coverage

- **60+ unit tests** for validation library functions
- **Workflow tests** for retry logic and command execution
- **Implementation tests** for phase detection and continuation prompts
- **Hook tests** for blocking/allowing behavior
- **Integration tests** for end-to-end workflows
- **Quickstart validation** to ensure documentation examples work

See `tests/README.md` for detailed testing documentation.

## Use Cases

### 1. Fully Automated Feature Development

```bash
# Start workflow with automatic validation
./scripts/speckit-workflow-validate.sh user-authentication

# Claude generates spec.md, plan.md, tasks.md with automatic retries
# If any file is missing, workflow automatically re-prompts Claude

# Check implementation status
./scripts/speckit-implement-validate.sh user-authentication

# Continue implementing until complete
# Validator provides specific continuation prompts
```

### 2. Hook-Enforced Completeness

```bash
# Enable implementation hook
claude --settings .claude/implement-hook-settings.json

# Claude cannot stop until all tasks in tasks.md are checked
# Hook automatically retries up to 3 times
# Provides clear feedback about remaining work
```

### 3. CI/CD Integration

```bash
# Validate feature completion in CI pipeline
if ./scripts/speckit-implement-validate.sh my-feature --json | jq -e '.status == "COMPLETE"'; then
  echo "Feature implementation complete"
  exit 0
else
  echo "Feature implementation incomplete"
  ./scripts/speckit-implement-validate.sh my-feature  # Show details
  exit 1
fi
```

### 4. Custom Workflows

```bash
# Source the library for custom scripts
source scripts/lib/speckit-validation-lib.sh

# Use validation functions
if validate_file_exists "spec.md" "specs/my-feature"; then
  echo "Spec exists"
fi

# Manage retry state
increment_retry_count "my-feature" "specify"
count=$(get_retry_count "my-feature" "specify")

# Generate continuation prompts
prompt=$(generate_continuation_prompt "my-feature" "Phase 2" "8")
echo "$prompt"
```

## Configuration

### Retry Limits

Default: 3 attempts per command

Override in scripts:
```bash
./scripts/speckit-workflow-validate.sh --max-retries 5 my-feature
```

Or set environment variable:
```bash
export SPECKIT_MAX_RETRIES=5
./scripts/speckit-workflow-validate.sh my-feature
```

### Spec Location

Default: `specs/<spec-name>/`

Override:
```bash
export SPEC_BASE_DIR="/path/to/my/specs"
./scripts/speckit-workflow-validate.sh my-feature
```

### Exit Codes

All scripts follow consistent exit code conventions:

- `0`: Success
- `1`: Validation failed
- `2`: Retry limit exhausted
- `3`: Invalid arguments
- `4`: Missing dependencies

## Performance

Optimized for speed:
- **Workflow validation**: ~0.22s average
- **Implementation validation**: ~0.15s average
- **Hook validation**: ~0.08s average

All validations complete in under 1 second, well below the 5-second target.

## Dependencies

See [PREREQUISITES.md](PREREQUISITES.md) for complete installation instructions.

**Quick check:**
```bash
# Check if all required tools are installed
specify check

# Manual check
command -v specify git >/dev/null && echo "Required dependencies found"
```

## Troubleshooting

### "Command not found: specify" or "Command not found: jq"

See [PREREQUISITES.md](PREREQUISITES.md) for installation instructions.

### "Retry limit exhausted"

The script attempted to validate 3 times but the required file was never created.

**Solutions:**
1. Check Claude's output for errors
2. Verify the spec directory exists
3. Manually create the missing file
4. Increase retry limit: `--max-retries 5`

### "Hook blocks Claude from stopping"

This is expected behavior when artifacts are incomplete.

**Solutions:**
1. Review the hook's error message for specific issues
2. Complete the required artifacts manually
3. Check retry count: hooks stop blocking after 3 attempts
4. Temporarily disable hook by removing it from settings

### Tests fail with "bats: command not found"

Install bats-core following instructions in `tests/README.md`.

## Contributing

Contributions welcome! Please:

1. Run tests before submitting: `./tests/run-all-tests.sh`
2. Follow shellcheck standards: `shellcheck scripts/**/*.sh`
3. Add tests for new features
4. Update documentation

## License

MIT License - see LICENSE file for details

## Credits

Created as part of the SpecKit Validation Hooks feature for Claude Code.

Built with:
- Bash scripting
- bats-core testing framework
- jq for JSON processing
- Claude Code hook system

## Support

Issues and questions: https://github.com/yourusername/auto-claude-speckit/issues

Documentation: See `tests/README.md` for testing guide
