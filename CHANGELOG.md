# Changelog

All notable changes to autospec will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **BREAKING**: DAG runtime state now stored inline in dag.yaml instead of separate state files, with automatic migration from legacy `.autospec/state/dag-runs/` files
- **BREAKING**: `dag run` is now idempotent - running the same command again automatically resumes from existing state
- **BREAKING**: All dag commands now use workflow file path instead of opaque run-id (e.g., `dag merge workflow.yaml` instead of `dag merge <run-id>`)
- **BREAKING**: State files now keyed by workflow filename (e.g., `features/v1.yaml` stores state as `features-v1.yaml.state`)
- `dag validate` no longer treats missing spec folders as errors; they are now shown as informational notes since specs are created dynamically during `dag run`
- Renamed `autospec dag` command to `autospec waves` for task execution wave visualization
- Renamed `internal/dag/` package to `internal/taskgraph/` to reserve `dag` for multi-spec orchestration

### Added
- `dag commit` command for manual commit triggering with `--only`, `--dry-run` flags, post-execution commit verification with configurable retry (`dag.autocommit`, `dag.autocommit_retries`, `dag.autocommit_cmd`), and `dag merge` pre-flight check blocking merges when worktrees have uncommitted code or no commits ahead of target branch
- DAG log storage in XDG cache directory (`~/.cache/autospec/dag-logs/`) with `dag clean-logs` command for bulk cleanup and `--logs`/`--logs-only` flags for `dag cleanup`
- Human-readable branch names for DAG specs using `dag/<dag-id>/<spec-id>` format with automatic ID resolution from `dag.id`, `dag.name`, or workflow filename, plus collision detection with hash suffix fallback
- DAG layer staging with progressive merge propagation: each layer branches from previous layer's staging branch, with `dag.automerge` config and `--automerge`/`--no-automerge`/`--no-layer-staging` flags
- `dag run --fresh` flag to discard existing state and start fresh
- `dag run --only spec1,spec2` flag to run only specified specs (requires existing state)
- `dag run --clean` flag (with `--only`) to clean artifacts and reset state for specific specs

### Removed
- **BREAKING**: `dag resume` command removed - functionality now built into idempotent `dag run`
- **BREAKING**: `dag retry` command removed - use `dag run --only spec1 --clean` to retry specific specs

### Added
- `worktree create` flags: `--skip-setup` (skip setup script), `--skip-copy` (skip directory copying), and `--no-rollback` (preserve worktree on failure for debugging), plus validation checks and setup script timeout enforcement
- `dag merge` and `dag cleanup` commands for merging completed specs with AI-assisted conflict resolution, and cleaning up worktrees
- `dag run --parallel` flag for concurrent spec execution with configurable parallelism (`--max-parallel N`, default 4)
- `dag run --fail-fast` flag to abort all running specs on first failure (requires `--parallel`)
- `dag status [run-id]` command showing categorized spec states (completed with duration, running with current stage/task, pending with blocking deps, blocked with failed deps, failed with error)
- Real-time progress tracking during parallel DAG execution (X/Y specs complete)
- Graceful SIGINT handling with state preservation within 2 seconds
- Output multiplexing with `[spec-id]` prefixes for parallel spec output
- Dependency-aware scheduling: blocked specs marked when dependencies fail
- Race condition tests for parallel executor thread safety
- User documentation for parallel DAG execution (`docs/public/dag-parallel.md`)
- `dag watch` command to monitor DAG run with live-updating status table, real-time spec progress, and `--interval` flag for refresh rate
- `dag logs` command to stream or view log output for a spec with `--no-follow` (one-shot mode) and `--latest` (most recent run) flags
- `dag list` enhanced output format showing specs count and relative time since run started (e.g., "3 specs • started 5m ago")
- `dag.max_log_size` config option for log size management with automatic truncation to prevent excessive disk usage (default: 50MB)
- `dag.on_conflict`, `dag.base_branch`, `dag.max_spec_retries` config options for DAG execution behavior
- LogTailer for real-time log streaming using fsnotify file watching
- Timestamped log output for DAG spec execution with `[YYYY-MM-DD HH:MM:SS]` format
- TruncatingWriter for automatic log file size management during long-running specs
- Documentation for dag watch, logs, and list commands (`docs/public/dag-watch-logs.md`)
- `dag run` and `dag list` commands for sequential DAG workflow execution with spec dependency ordering
- `dag validate` and `dag visualize` commands for multi-spec DAG workflow validation with cycle detection, missing spec checks, and ASCII visualization
- Enhanced integration tests with MockExecutor argument/env capture, ArgumentValidator for CLI flags, and TestHelperProcess pattern for zero-API-call testing
- Verification config block with tiered validation levels (basic/enhanced/full), feature toggles, and quality thresholds
- EARS (Easy Approach to Requirements Syntax) support in spec.yaml with pattern-specific validation and instruction injection

### Fixed
- Data race in `ParallelExecutor.markRunning` and `markDone` when updating `state.RunningCount` concurrently
- Data race in `ParallelExecutor.handleInterruption` during YAML serialization of state

## [0.9.0] - 2026-01-16

### Added
- `autospec init` now supports non-interactive mode with flags: `--sandbox`, `--skip-permissions`, `--gitignore`, `--constitution`, and `--use-subscription` (each with `--no-*` counterpart) for CI/CD automation
- `autospec init` now creates `.autospec/init.yml` to track initialization settings (scope, agent, version) for accurate doctor checks
- Core git operations now use go-git library internally, reducing dependency on git CLI for branch detection, repository root finding, and remote fetching
- `autospec doctor` no longer checks for git CLI installation (git CLI still required for worktree commands)
- `autospec prereqs` now outputs `IS_GIT_REPO` field; implement template uses this instead of git CLI for repo detection

### Fixed
- `autospec doctor` now checks global agent settings when `init.yml` indicates global scope was used during init

## [0.8.2] - 2026-01-05

### Added
- `skip_permissions` config option to pass `--dangerously-skip-permissions` flag to autospec Claude runs (does not modify Claude settings)
- `autospec init` now prompts to configure `skip_permissions` (recommended: Yes) for autonomous Claude runs; skips prompt if already enabled

## [0.8.1] - 2026-01-03

### Fixed
- `autospec init` now correctly installs `.claude/commands/` slash command templates when Claude is selected as an agent

## [0.8.0] - 2026-01-03

### Added
- OpenCode agent preset now fully functional with `autospec init --ai opencode` or `agent_preset: opencode` in config
- `autospec init [path]` now accepts an optional path argument to initialize projects at specified locations (e.g., `autospec init ~/projects/myapp`)
- Colored output formatting with clear visual markers to distinguish agent output from autospec status messages
- Interactive agent selection during `autospec init` - choose between Claude Code and OpenCode
- `config sync` command to synchronize configuration with current schema (adds new options with defaults, removes deprecated options)
- `config toggle` command to toggle boolean configuration values
- `config keys` command to list all available configuration keys with types and descriptions
- Automatic config sync after `autospec update` - new config options are added and deprecated ones removed while preserving user settings

### Fixed
- Implement template now has explicit "Execution Boundaries" section preventing agents from continuing past `--phase N` scope
- Security notice about `--dangerously-skip-permissions` now only shows for Claude agent (skipped for OpenCode, Gemini, etc.)
- `skip_permissions_notice_shown` config key now properly recognized and persists after first display
- OpenCode `permission.edit` now correctly uses simple string format (`"allow"`) instead of object with patterns, fixing `Invalid option: expected one of "ask"|"allow"|"deny"` error

### Changed
- Agent permissions now write to global config by default (`~/.claude/settings.json`, `~/.config/opencode/opencode.json`); use `--project` for project-level
- **BREAKING**: Consolidated `output_style` config into `cclean.style` - run `autospec config sync` after upgrading
- `autospec init` no longer prompts about git worktrees; shows info message with `autospec worktree gen-script` command instead
- Risk assessment in `plan` stage now opt-in (disabled by default); enable with `autospec config set enable_risk_assessment true`
- Reorganized `docs/` into `public/` (user-facing) and `internal/` (contributor) subdirectories
- Site generation now automated via GitHub Actions - generated files no longer committed to git
- Added `make docs-sync` and updated `make serve` to auto-sync docs before serving

## [0.7.3] - 2025-12-21

### Changed
- `auto_commit` now defaults to `false` (was `true`) - use `--auto-commit` flag or set `auto_commit: true` in config to enable

## [0.7.2] - 2025-12-21

### Fixed
- Add missing `auto_commit` field to default config template generated by `autospec init`
- `update` command now handles cross-device moves (e.g., `/tmp` to `/home`) by falling back to copy when `rename` fails with `EXDEV`

## [0.7.1] - 2025-12-21

### Added
- Interactive mode defaults for `clarify` and `analyze` commands; notification when interactive session starts after automated stages in `run` command
- Process replacement via `syscall.Exec` for interactive stages to ensure full terminal control in TUI applications

### Fixed
- Hide help menu on workflow execution errors; still shown for incorrect command usage
- Spec status validation now accepts `Completed` instead of `Implemented` as final status

## [0.7.0] - 2025-12-21

### Added
- `--auto-commit` and `--no-auto-commit` flags for automatic git commit creation after workflow completion with conventional commit messages
- Compact auto-commit output display (`[+AutoCommit]` tag) and minimal agent instructions (~15 lines vs ~90)
- `auto_commit` config option for automatic git commits after workflow completion (overridable via CLI flags)
- `update` command for self-updating autospec to the latest GitHub release with SHA256 checksum verification, automatic backup, and atomic installation with rollback on failure
- `ck` (check) command to quickly check for newer versions on GitHub releases

### Removed
- Async update check from `version` command (moved to dedicated `ck` command)
- `handoffs` frontmatter field from command templates (was non-functional)

## [0.6.1] - 2025-12-20

### Added
- `use_subscription` config option (default: `true`) to force Claude subscription mode and prevent accidental API charges; auto-detected during `init`
- Use output-format stream-json mode by default for claude sessions

## [0.6.0] - 2025-12-20

### Changed
- Multi-agent support (in development) now gated to dev builds only; production builds default to Claude Code
- DAG-based parallel execution (in development) gated to dev builds only
- `init` command now collects all user choices before applying changes, with final confirmation before running Claude sessions

### Added
- **[Dev builds only]** DAG-based parallel task execution with `implement --parallel` flag for concurrent task processing
- **[Dev builds only]** `--max-parallel` flag to limit concurrent task execution (default: number of CPU cores)
- **[Dev builds only]** `--worktrees` flag for git worktree-based task isolation during parallel execution
- **[Dev builds only]** `--dry-run` flag to preview execution plan without running tasks
- **[Dev builds only]** `--yes` flag to skip resume confirmation prompts
- **[Dev builds only]** `dag` command to visualize task dependencies as ASCII graph with wave grouping
- **[Dev builds only]** Parallel execution state persistence with resume support (R/W/S/A options: resume, resume wave, skip failed, abort)
- Multi-agent CLI abstraction layer with 6 built-in agents (claude, cline, gemini, codex, opencode, goose) and custom agent support via `agent_preset` config or `--agent` flag
- Structured `custom_agent` config with explicit `command`, `args`, `env`, and `post_processor` fields (replaces error-prone shell string parsing)
- Agent discovery and status in `autospec doctor` showing installed agents with versions
- `view` command to display dashboard overview of all specs with completion status and task progress
- `worktree` command for git worktree management (create, list, remove, setup, prune) with automatic project setup
- `worktree gen-script` command to generate project-specific setup scripts for worktrees
- `init` command now prompts to create constitution if none exists (Y/n default yes)
- `init` command now prompts to generate worktree setup script if not already present (y/N default no)
- Dark mode support for GitHub Pages documentation site
- `init` command now displays permissions/sandbox configuration status and prompts to configure sandbox if not set up
- `init` command shows recommended full automation setup with cclean post_processor and --dangerously-skip-permissions disclaimer on first run
- Native cclean (claude-clean) library integration as internal dependency for beautiful Claude JSONL output parsing with `--output-style` flag and `output_style` config option
- One-time security notice on first workflow run explaining `--dangerously-skip-permissions` usage with sandbox status; suppress via `AUTOSPEC_SKIP_PERMISSIONS_NOTICE=1`
- `init` command now prompts to add `.autospec/` to `.gitignore` with guidance for shared vs personal repos

### Changed
- `init` constitution prompt now explains it's a one-time setup that defines project coding standards
- `init` agent selection now uses interactive arrow-key navigation with space to toggle (replaces number input)

### Removed
- **BREAKING**: Removed legacy config fields `claude_cmd`, `claude_args`, `custom_claude_cmd` (use `agent_preset` or structured `custom_agent` instead)

## [0.5.0] - 2025-12-18

### Added
- `config set/get/toggle/keys` subcommands for CLI-based configuration management with `--user` and `--project` scope flags
- `--max-retries, -r` flag for `plan`, `tasks`, `constitution`, and `checklist` commands to override config retry limit

### Changed
- Improved internal codebase structure for faster future development and better reliability

### Fixed
- Description propagation in `run -a` now matches `autospec all` behavior (only specify stage receives description)

## [0.4.0] - 2025-12-18

### Added
- GitHub Pages documentation website with architecture overview, internals guide, FAQ, and troubleshooting pages
- `ContextMeta` struct to reduce redundant artifact file reads during phase execution
- `task block` and `task unblock` commands to mark tasks as blocked with documented reasons
- `BlockedReason` field in tasks.yaml to track why tasks are blocked (with validation warnings when missing)
- `risks` section in plan.yaml for documenting implementation risks and mitigation strategies
- Schema validation for YAML artifacts (validates structure, not just existence)
- `notes` field in tasks.yaml for additional task context (max 1000 chars)

### Changed
- CLI commands reorganized into subpackages (`stages/`, `config/`, `util/`, `admin/`, `shared/`) for improved maintainability
- Documentation restructured into feature cards for better presentation
- Custom sidebar styles for improved layout and usability
- Pre-flight validation now distinguishes between missing and invalid artifacts with specific error messages
- Retry state resets automatically when starting the specify stage

### Fixed
- Retry context instructions now dynamically injected only during retries (reduces token waste on first-run executions)
- Improved artifact validation shows both missing and invalid files in error output

## [0.3.2] - 2025-12-17

### Added
- `sauce` command to display the project source URL

### Changed
- Installer shows download progress bar for better visibility
- Default installation directory changed to `~/.local/bin`
- Installer now backs up existing binary before upgrading

### Fixed
- Improved installer reliability with better error handling and temp file cleanup
- Fixed POSIX compatibility issues in installer color output

## [0.3.1] - 2025-12-16

### Added
- ASCII art logo in installer

### Changed
- Installer uses `sh` instead of `bash` for better compatibility

## [0.3.0] - 2025-12-16

### Added

- `history` command with two-phase logging, status tracking, and `--status` filter
- Cross-platform notifications for command/stage completion (macOS, Linux)
- Claude settings validation and automatic permission configuration
- Profile management system for configuration presets
- Lifecycle wrapper for CLI commands (timing, notifications, history)
- Context injection for phase execution (performance optimization)
- Task-level execution mode with `--tasks` and `--from-task` flags
- `--single-session` flag for legacy single-session execution
- `--from-phase` and `--phase` flags for phase-level control
- `implement_method` config option for default execution mode
- Prerequisite validation for CLI commands (pre-flight artifact checks)
- Artifact validation for analysis, checklist, and constitution YAML files
- Optional stage commands: `constitution`, `clarify`, `checklist`, `analyze`
- `run` command with stage selection flags (`-s`, `-p`, `-t`, `-i`, `-a`, `-n`, `-r`, `-l`, `-z`)
- `--dry-run` flag for previewing actions
- `--debug` flag for verbose logging
- `update-task` command for task status management
- Spec status tracking with automatic completion marking
- `skip_confirmations` config and `AUTOSPEC_YES` environment variable
- `config migrate` command for config file migration
- Custom Claude command support with `{{PROMPT}}` placeholder
- claude-clean integration for readable streaming output
- Auto-updates to spec.yaml and tasks.yaml during execution
- Phase-isolated sessions (80%+ cost savings on large specs)
- Quickstart guide with interactive demo script
- Internals documentation guide
- Checklists documentation for requirement validation
- Shell completion support (bash, zsh, fish)

### Changed

- Renamed "phase" to "stage" throughout codebase for clarity
- Dropped Windows support; WSL recommended
- Long-running notification threshold: 30s → 2 minutes
- Renamed `full` command to `all`
- Refactored tests to map-based table-driven pattern
- Improved error handling with context wrapping

### Fixed

- Constitution requirement checks across all commands
- Task status tracking during implementation
- Artifact dependency validation
- Claude settings configuration in `init` command

## [0.2.0] - 2025-01-15

### Added

- Workflow progress indicators with spinners
- Command execution timeout support
- Timeout configuration via `AUTOSPEC_TIMEOUT` environment variable
- Exit code 5 for timeout errors
- Configurable timeout in config files (0 for infinite, 1-604800 seconds)

### Changed

- Enhanced workflow orchestration with better error handling
- Improved phase execution with clearer status messages

## [0.1.0] - 2025-01-01

### Added

- Initial Go binary implementation
- CLI commands: `workflow`, `specify`, `plan`, `tasks`, `implement`, `status`, `init`, `config`, `doctor`, `version`
- Cobra-based command structure with global flags
- Workflow orchestration (specify -> plan -> tasks -> implement)
- Hierarchical configuration system using koanf
- Configuration sources: environment variables, local config, global config, defaults
- Retry management with persistent state tracking
- Atomic file writes for retry state consistency
- Validation system with <10ms performance contract
- Spec detection from git branch or most recently modified directory
- Git integration helpers
- Pre-flight dependency checks (claude, specify CLIs)
- Claude execution modes: CLI, API, and custom command
- Custom command support with `{{PROMPT}}` placeholder
- Exit code conventions for programmatic use
- Cross-platform builds (Linux, macOS, Windows)

### Changed

- Migrated from bash scripts to Go binary
- Replaced manual validation with automated checks

### Deprecated

- Legacy bash scripts in `scripts/` (scheduled for removal)
- Bats tests in `tests/` (being replaced by Go tests)

[Unreleased]: https://github.com/ariel-frischer/autospec/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/ariel-frischer/autospec/compare/v0.8.2...v0.9.0
[0.8.2]: https://github.com/ariel-frischer/autospec/compare/v0.8.1...v0.8.2
[0.8.1]: https://github.com/ariel-frischer/autospec/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/ariel-frischer/autospec/compare/v0.7.3...v0.8.0
[0.7.3]: https://github.com/ariel-frischer/autospec/compare/v0.7.2...v0.7.3
[0.7.2]: https://github.com/ariel-frischer/autospec/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/ariel-frischer/autospec/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/ariel-frischer/autospec/compare/v0.6.1...v0.7.0
[0.6.1]: https://github.com/ariel-frischer/autospec/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/ariel-frischer/autospec/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/ariel-frischer/autospec/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/ariel-frischer/autospec/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/ariel-frischer/autospec/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/ariel-frischer/autospec/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/ariel-frischer/autospec/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/ariel-frischer/autospec/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/ariel-frischer/autospec/releases/tag/v0.1.0
