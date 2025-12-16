// Package claude provides Claude Code settings management for autospec.
// It handles loading, validating, and modifying .claude/settings.local.json
// files to ensure Claude Code has the necessary permissions to execute
// autospec commands.
//
// The package supports:
//   - Loading and parsing Claude settings files
//   - Checking if required permissions are configured
//   - Adding permissions while preserving existing settings
//   - Detecting deny list conflicts
//   - Atomic file writes to prevent corruption
package claude
