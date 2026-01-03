// Package opencode provides OpenCode settings management for autospec.
// It handles loading, validating, and modifying opencode.json
// files to ensure OpenCode has the necessary permissions to execute
// autospec commands.
//
// The package supports:
//   - Loading and parsing OpenCode settings files (opencode.json)
//   - Checking if required permissions are configured
//   - Adding bash permissions while preserving existing settings
//   - Atomic file writes to prevent corruption
//
// OpenCode stores its configuration in opencode.json at the project root,
// unlike Claude which uses .claude/settings.local.json.
package opencode
