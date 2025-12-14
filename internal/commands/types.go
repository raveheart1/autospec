// Package commands provides embedded command templates and template management.
package commands

// CommandTemplate represents an embedded command template.
type CommandTemplate struct {
	Name        string // Filename without extension (e.g., "autospec.specify")
	Description string // Description from YAML frontmatter
	Version     string // Template schema version
	Content     []byte // Raw markdown content
}

// VersionMismatch represents a version comparison result.
type VersionMismatch struct {
	CommandName      string // Name of the command
	InstalledVersion string // Version in installed file (empty if not installed)
	EmbeddedVersion  string // Version in embedded template
	Action           string // "update" or "install"
}

// InstallResult represents the result of installing a command template.
type InstallResult struct {
	CommandName string // Name of the command that was installed
	Action      string // "installed", "updated", or "skipped"
	Path        string // Path where the command was installed
}

// CommandInfo represents information about an installed command.
type CommandInfo struct {
	Name             string // Command name
	Description      string // Description from frontmatter
	Version          string // Installed version
	EmbeddedVersion  string // Version in embedded template
	IsOutdated       bool   // True if installed version differs from embedded
	InstallPath      string // Full path to installed file
}
