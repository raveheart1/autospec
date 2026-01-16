package init

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// SchemaVersion is the current version of the init.yml schema.
// Increment this when making breaking changes to the schema.
const SchemaVersion = "1.0.0"

// DefaultFileName is the name of the init settings file.
const DefaultFileName = "init.yml"

// Scope values for SettingsScope field.
const (
	ScopeGlobal  = "global"
	ScopeProject = "project"
)

// Settings represents the contents of .autospec/init.yml.
// This file tracks how autospec was initialized in a project.
type Settings struct {
	// Version is the schema version for future compatibility.
	Version string `yaml:"version"`

	// SettingsScope indicates where agent permissions were written.
	// Valid values: "global" or "project".
	SettingsScope string `yaml:"settings_scope"`

	// AutospecVersion is the version of autospec that ran init.
	AutospecVersion string `yaml:"autospec_version"`

	// Agents lists the configured agents and their settings.
	Agents []AgentEntry `yaml:"agents,omitempty"`

	// CreatedAt is when init.yml was first created.
	CreatedAt time.Time `yaml:"created_at"`

	// UpdatedAt is when init.yml was last modified.
	UpdatedAt time.Time `yaml:"updated_at"`
}

// AgentEntry represents configuration for a single agent within init.yml.
type AgentEntry struct {
	// Name is the agent identifier (e.g., "claude", "opencode").
	Name string `yaml:"name"`

	// Configured indicates whether the agent was successfully configured.
	Configured bool `yaml:"configured"`

	// SettingsFile is the path to the agent's settings file that was modified.
	SettingsFile string `yaml:"settings_file"`
}

// DefaultPath returns the default path for init.yml relative to project root.
// The path is .autospec/init.yml.
func DefaultPath() string {
	return filepath.Join(".autospec", DefaultFileName)
}

// Exists checks if init.yml exists at the default location.
func Exists() bool {
	_, err := os.Stat(DefaultPath())
	return err == nil
}

// ExistsAt checks if init.yml exists at the given path.
func ExistsAt(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Load reads and parses init.yml from the default location.
// Returns an error if the file doesn't exist or is invalid YAML.
func Load() (*Settings, error) {
	return LoadFrom(DefaultPath())
}

// LoadFrom reads and parses init.yml from the given path.
// Returns an error if the file doesn't exist or is invalid YAML.
func LoadFrom(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading init settings file: %w", err)
	}

	var settings Settings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing init settings YAML: %w", err)
	}

	return &settings, nil
}

// Save writes the Settings to the default init.yml location.
// Creates the parent directory if it doesn't exist.
func (s *Settings) Save() error {
	return s.SaveTo(DefaultPath())
}

// SaveTo writes the Settings to the given path.
// Creates the parent directory if it doesn't exist.
func (s *Settings) SaveTo(path string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating init settings directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling init settings: %w", err)
	}

	// Write file with restrictive permissions
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing init settings file: %w", err)
	}

	return nil
}

// NewSettings creates a new Settings with default values.
// The caller should set SettingsScope and Agents based on init options.
func NewSettings(autospecVersion string) *Settings {
	now := time.Now()
	return &Settings{
		Version:         SchemaVersion,
		AutospecVersion: autospecVersion,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// IsValidScope returns true if the given scope is a valid value.
func IsValidScope(scope string) bool {
	return scope == ScopeGlobal || scope == ScopeProject
}
