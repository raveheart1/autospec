package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SettingsStatus represents the state of OpenCode settings configuration.
type SettingsStatus int

const (
	// StatusConfigured indicates the settings file exists with the required permission.
	StatusConfigured SettingsStatus = iota
	// StatusMissing indicates the settings file does not exist.
	StatusMissing
	// StatusNeedsPermission indicates the settings file exists but lacks the required permission.
	StatusNeedsPermission
	// StatusDenied indicates the required permission is explicitly denied.
	StatusDenied
)

// String returns a human-readable representation of the status.
func (s SettingsStatus) String() string {
	switch s {
	case StatusConfigured:
		return "Configured"
	case StatusMissing:
		return "Missing"
	case StatusNeedsPermission:
		return "NeedsPermission"
	case StatusDenied:
		return "Denied"
	default:
		return "Unknown"
	}
}

// SettingsCheckResult contains the result of validating OpenCode settings.
type SettingsCheckResult struct {
	Status   SettingsStatus
	Message  string
	FilePath string
}

// Permission levels for OpenCode bash permissions.
const (
	PermissionAllow = "allow"
	PermissionAsk   = "ask"
	PermissionDeny  = "deny"
)

// RequiredPattern is the bash permission pattern autospec needs in OpenCode settings.
const RequiredPattern = "autospec *"

// SettingsFileName is the name of the OpenCode settings file.
const SettingsFileName = "opencode.json"

// Permission represents the permission configuration in opencode.json.
type Permission struct {
	Bash map[string]string `json:"bash,omitempty"`
}

// Settings represents an OpenCode settings file with flexible JSON structure.
// Uses a combination of typed fields and raw JSON to preserve unknown fields.
type Settings struct {
	Permission Permission `json:"permission,omitempty"`

	// extra holds unknown fields to preserve during save
	extra map[string]json.RawMessage

	filePath string
}

// NewSettings creates a new empty Settings instance for the given project directory.
func NewSettings(projectDir string) *Settings {
	return &Settings{
		Permission: Permission{
			Bash: make(map[string]string),
		},
		extra:    make(map[string]json.RawMessage),
		filePath: filepath.Join(projectDir, SettingsFileName),
	}
}

// Load reads and parses OpenCode settings from the project directory.
// Returns a Settings instance even if the file doesn't exist (with empty data).
// Returns an error only for actual failures like permission errors or malformed JSON.
func Load(projectDir string) (*Settings, error) {
	settingsPath := filepath.Join(projectDir, SettingsFileName)
	return loadFromPath(settingsPath)
}

// loadFromPath loads settings from a specific file path.
func loadFromPath(settingsPath string) (*Settings, error) {
	s := &Settings{
		Permission: Permission{
			Bash: make(map[string]string),
		},
		extra:    make(map[string]json.RawMessage),
		filePath: settingsPath,
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading settings file %s: %w", settingsPath, err)
	}

	if len(data) == 0 {
		return s, nil
	}

	if err := s.unmarshalPreservingExtra(data); err != nil {
		return nil, fmt.Errorf("parsing settings file %s: %w", settingsPath, err)
	}

	return s, nil
}

// unmarshalPreservingExtra unmarshals JSON while preserving unknown fields.
func (s *Settings) unmarshalPreservingExtra(data []byte) error {
	// First, unmarshal everything into a raw map
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract and parse the permission field
	if permRaw, ok := raw["permission"]; ok {
		if err := json.Unmarshal(permRaw, &s.Permission); err != nil {
			return fmt.Errorf("parsing permission field: %w", err)
		}
		delete(raw, "permission")
	}

	// Ensure Bash map is initialized
	if s.Permission.Bash == nil {
		s.Permission.Bash = make(map[string]string)
	}

	// Store remaining fields as extra
	s.extra = raw

	return nil
}

// FilePath returns the path to the settings file.
func (s *Settings) FilePath() string {
	return s.filePath
}

// Exists returns true if the settings file exists on disk.
func (s *Settings) Exists() bool {
	_, err := os.Stat(s.filePath)
	return err == nil
}

// CheckBashPermission returns the permission level for a given pattern.
// Returns empty string if the pattern is not configured.
func (s *Settings) CheckBashPermission(pattern string) string {
	if s.Permission.Bash == nil {
		return ""
	}
	return s.Permission.Bash[pattern]
}

// AddBashPermission adds or updates a bash permission rule.
// This is idempotent - calling multiple times with the same pattern and level has no additional effect.
func (s *Settings) AddBashPermission(pattern, level string) {
	if s.Permission.Bash == nil {
		s.Permission.Bash = make(map[string]string)
	}
	s.Permission.Bash[pattern] = level
}

// HasRequiredPermission checks if the autospec permission is properly configured.
func (s *Settings) HasRequiredPermission() bool {
	level := s.CheckBashPermission(RequiredPattern)
	return level == PermissionAllow
}

// IsPermissionDenied checks if the autospec permission is explicitly denied.
func (s *Settings) IsPermissionDenied() bool {
	level := s.CheckBashPermission(RequiredPattern)
	return level == PermissionDeny
}

// Check validates OpenCode settings for the required autospec permission.
// Returns a SettingsCheckResult with appropriate status and message.
func (s *Settings) Check() SettingsCheckResult {
	if !s.Exists() {
		return SettingsCheckResult{
			Status:   StatusMissing,
			Message:  "opencode.json not found (run 'autospec init --ai opencode' to configure)",
			FilePath: "",
		}
	}

	if s.IsPermissionDenied() {
		return SettingsCheckResult{
			Status:   StatusDenied,
			Message:  fmt.Sprintf("'%s' permission is explicitly denied in %s", RequiredPattern, s.filePath),
			FilePath: s.filePath,
		}
	}

	if !s.HasRequiredPermission() {
		return SettingsCheckResult{
			Status:   StatusNeedsPermission,
			Message:  fmt.Sprintf("missing '%s': 'allow' permission (run 'autospec init --ai opencode' to fix)", RequiredPattern),
			FilePath: s.filePath,
		}
	}

	return SettingsCheckResult{
		Status:   StatusConfigured,
		Message:  fmt.Sprintf("'%s': 'allow' permission configured", RequiredPattern),
		FilePath: s.filePath,
	}
}

// CheckInDir performs a settings check for the given project directory.
// This is a convenience function that loads settings and checks them in one call.
func CheckInDir(projectDir string) (SettingsCheckResult, error) {
	settings, err := Load(projectDir)
	if err != nil {
		return SettingsCheckResult{}, fmt.Errorf("loading opencode settings: %w", err)
	}
	return settings.Check(), nil
}

// Save writes the settings to disk using atomic write (temp file + rename).
// Preserves unknown fields that were present when the file was loaded.
// Written JSON is pretty-printed with indentation for human readability.
func (s *Settings) Save() error {
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	data, err := s.marshalWithExtra()
	if err != nil {
		return fmt.Errorf("serializing settings: %w", err)
	}

	return atomicWrite(s.filePath, data)
}

// marshalWithExtra marshals the settings while preserving extra fields.
func (s *Settings) marshalWithExtra() ([]byte, error) {
	// Start with extra fields
	result := make(map[string]interface{})
	for k, v := range s.extra {
		var val interface{}
		if err := json.Unmarshal(v, &val); err != nil {
			return nil, fmt.Errorf("unmarshaling extra field %s: %w", k, err)
		}
		result[k] = val
	}

	// Add permission if it has content
	if len(s.Permission.Bash) > 0 {
		result["permission"] = s.Permission
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}

	// Add trailing newline for POSIX compliance
	data = append(data, '\n')
	return data, nil
}

// atomicWrite writes data to a file atomically using temp file + rename.
func atomicWrite(filePath string, data []byte) error {
	dir := filepath.Dir(filePath)
	tmpFile, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on any error
	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("renaming temp file to %s: %w", filePath, err)
	}

	// Clear tmpPath so defer doesn't try to remove the final file
	tmpPath = ""
	return nil
}
