package cliagent

// Configurator is an optional interface that agents can implement to provide
// project-level setup and configuration. Agents implementing this interface
// can configure settings files, permissions, and other project-specific
// resources during autospec init.
type Configurator interface {
	// ConfigureProject sets up the agent for a specific project.
	// It configures settings files, permissions, and other project-specific
	// resources. The method should be idempotent - calling it multiple times
	// with the same arguments should produce the same result.
	//
	// Parameters:
	//   - projectDir: The root directory of the project
	//   - specsDir: The directory where specs are stored (e.g., "specs" or "features")
	//   - projectLevel: If true, writes to project-level config; if false, writes to
	//     global config (for agents that support global configuration like OpenCode)
	//
	// Returns:
	//   - ConfigResult describing what was configured
	//   - error if configuration failed
	ConfigureProject(projectDir, specsDir string, projectLevel bool) (ConfigResult, error)
}

// ConfigResult describes the outcome of agent project configuration.
type ConfigResult struct {
	// PermissionsAdded lists permissions that were added during configuration.
	// Empty if no permissions were added (e.g., already configured).
	PermissionsAdded []string

	// AlreadyConfigured is true if all required configuration was already present.
	// When true, PermissionsAdded will typically be empty.
	AlreadyConfigured bool

	// Warning contains an optional warning message, such as when a deny list
	// entry conflicts with required permissions.
	Warning string
}

// SandboxResult describes the outcome of sandbox configuration.
type SandboxResult struct {
	// PathsAdded lists paths that were added to additionalAllowWritePaths.
	PathsAdded []string
	// ExistingPaths lists paths that were already configured.
	ExistingPaths []string
	// AlreadyConfigured is true if all required configuration was already present
	// (sandbox enabled AND all paths present).
	AlreadyConfigured bool
	// SandboxEnabled indicates if sandbox is enabled in settings (after configuration).
	SandboxEnabled bool
	// SandboxWasEnabled is true if sandbox was enabled during this configuration
	// (i.e., it was disabled before and we enabled it).
	SandboxWasEnabled bool
}

// SandboxConfigurator is an optional interface that agents can implement to
// configure sandbox write paths for secure execution.
type SandboxConfigurator interface {
	// GetSandboxPaths returns the paths required for sandbox write access.
	GetSandboxPaths(specsDir string) []string
	// ConfigureSandbox adds the required paths to the sandbox configuration.
	// Returns SandboxResult describing what was configured.
	ConfigureSandbox(projectDir, specsDir string) (SandboxResult, error)
}

// Configure checks if the given agent implements Configurator and calls
// ConfigureProject if it does. Returns nil, nil if the agent does not
// implement Configurator.
//
// The projectLevel parameter determines where configuration is written:
//   - true: writes to project-level config (e.g., ./opencode.json)
//   - false: writes to global config (e.g., ~/.config/opencode/opencode.json)
//
// Note: Some agents like Claude always use project-level config regardless of this flag.
func Configure(agent Agent, projectDir, specsDir string, projectLevel bool) (*ConfigResult, error) {
	configurator, ok := agent.(Configurator)
	if !ok {
		return nil, nil
	}

	result, err := configurator.ConfigureProject(projectDir, specsDir, projectLevel)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// IsConfigurator returns true if the given agent implements the Configurator
// interface.
func IsConfigurator(agent Agent) bool {
	_, ok := agent.(Configurator)
	return ok
}

// IsSandboxConfigurator returns true if the given agent implements the
// SandboxConfigurator interface.
func IsSandboxConfigurator(agent Agent) bool {
	_, ok := agent.(SandboxConfigurator)
	return ok
}

// ConfigureSandbox checks if the given agent implements SandboxConfigurator and
// calls ConfigureSandbox if it does. Returns nil, nil if the agent does not
// implement SandboxConfigurator.
func ConfigureSandbox(agent Agent, projectDir, specsDir string) (*SandboxResult, error) {
	configurator, ok := agent.(SandboxConfigurator)
	if !ok {
		return nil, nil
	}

	result, err := configurator.ConfigureSandbox(projectDir, specsDir)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSandboxPaths returns the sandbox paths for an agent if it implements
// SandboxConfigurator, nil otherwise.
func GetSandboxPaths(agent Agent, specsDir string) []string {
	configurator, ok := agent.(SandboxConfigurator)
	if !ok {
		return nil
	}
	return configurator.GetSandboxPaths(specsDir)
}
