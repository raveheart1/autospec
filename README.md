# Auto Claude SpecKit

Automated validation and workflow scripts for Claude Code's SpecKit feature development system.

## What is This?

Auto Claude SpecKit is a cross-platform Go binary and Claude Code hook system that automates the validation of SpecKit workflows. It ensures your feature specifications, plans, and tasks are complete before allowing Claude to stop, and automatically retries commands when artifacts are missing.

### Key Features

- **Automated Workflow Validation**: Runs `/speckit` commands with automatic retry when outputs are missing
- **Hook-Based Enforcement**: Prevents Claude from stopping until required artifacts exist
- **Phase Completion Detection**: Validates implementation progress in `tasks.md`
- **Continuation Prompts**: Automatically generates prompts to resume incomplete work
- **Performance Optimized**: Sub-second validation times (<10ms per validation)
- **Cross-Platform**: Native binaries for Linux, macOS, and Windows
- **Comprehensive Testing**: Unit tests, benchmarks, and integration tests in Go

## Quick Start

### Installation

#### Prerequisites

**Required:**
- Claude Code CLI (see https://www.claude.com/product/claude-code)
- SpecKit CLI (`uv tool install specify-cli --from git+https://github.com/github/spec-kit.git`)
- Git
- Go 1.21+ (for building from source)

**Optional:**
- jq (JSON processor for config manipulation)
- make (for using Makefile commands)

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

**Binary size**: ~9.7 MB per platform
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
make build

# Or build for all platforms (Linux/macOS/Windows)
make build-all

# Install locally
make install

# Verify installation
autospec version
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

#### 1. First-Time Setup

```bash
# Create project configuration
autospec init

# Initialize SpecKit templates (if not already done)
specify init . --ai claude --force
```

#### 2. Run Complete Workflow

The simplest way to generate a complete feature specification:

```bash
autospec workflow "Add user authentication with OAuth support"
```

This automatically:
- Generates `specs/<feature-name>/spec.md` (specification)
- Creates `specs/<feature-name>/plan.md` (implementation plan)
- Produces `specs/<feature-name>/tasks.md` (actionable tasks)
- Validates each artifact before proceeding
- Retries up to 3 times if any file is missing

#### 3. Check Progress and Continue

```bash
# Check current implementation status
autospec status

# Continue implementing remaining tasks
autospec implement
```

#### 4. Advanced Options

```bash
# Custom retry limit
autospec workflow "feature description" --max-retries 5

# Dry-run mode (see what would execute)
autospec workflow "feature" --dry-run

# Debug mode with verbose logging
autospec --debug workflow "feature"

# View current configuration
autospec config show
```

#### 5. Hook-Based Automatic Validation

Enable hooks to prevent Claude from stopping until artifacts are complete. See [CONTRIBUTORS.md](CONTRIBUTORS.md) for details on hook configuration.

## Use Cases

### 1. Fully Automated Feature Development

```bash
# Start workflow with automatic validation
autospec workflow "Add user authentication with OAuth"

# Claude generates spec.md, plan.md, tasks.md with automatic retries
# If any file is missing, workflow automatically re-prompts Claude

# Check implementation status
autospec status

# Continue implementing remaining tasks
autospec implement
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
autospec status --json | jq -e '.status == "COMPLETE"'
if [ $? -eq 0 ]; then
  echo "Feature implementation complete"
  exit 0
else
  echo "Feature implementation incomplete"
  autospec status  # Show details
  exit 1
fi
```

### 4. Working with Configuration

```bash
# Initialize project configuration
autospec init

# View current configuration
autospec config show

# Use custom specs directory
autospec --specs-dir ./features workflow "new feature"

# Adjust retry limits
autospec workflow "feature" --max-retries 5
```

## Configuration

Autospec uses configuration files for customization:

- **Local config**: `.autospec/config.json` (project-specific)
- **Global config**: `~/.autospec/config.json` (user-wide)
- **Environment variables**: `AUTOSPEC_*` prefix

Common settings:
```bash
# Initialize config
autospec init

# View current config
autospec config show

# Use environment variables
export AUTOSPEC_MAX_RETRIES=5
export AUTOSPEC_SPECS_DIR="./features"
```

See [CONTRIBUTORS.md](CONTRIBUTORS.md) for detailed configuration options.

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

### Build fails with "Go not found"

Install Go 1.21+ from https://go.dev/doc/install

Verify installation: `go version`

## Contributing

Contributions welcome! Please:

1. Run tests before submitting: `make test`
2. Follow Go standards: `make lint`
3. Add tests for new features (table-driven tests preferred)
4. Add benchmarks for performance-critical code
5. Update documentation (README.md and CLAUDE.md)

## License

MIT License - see LICENSE file for details

## Credits

Created as part of the SpecKit Validation Hooks feature for Claude Code.

Built with:
- Go 1.21+ (cross-platform binary)
- Cobra CLI framework
- Koanf configuration library
- Claude Code hook system

## Support

Issues and questions: https://github.com/anthropics/auto-claude-speckit/issues

Documentation:
- See `CLAUDE.md` for development guide
- See `Makefile` for available commands
- Run `autospec --help` for CLI usage
