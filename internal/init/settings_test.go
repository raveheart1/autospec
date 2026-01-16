// Package init_test tests the init settings management for autospec.
// Related: internal/init/settings.go
// Tags: init, settings, yaml, persistence
package init

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPath(t *testing.T) {
	t.Parallel()
	path := DefaultPath()
	assert.Equal(t, filepath.Join(".autospec", "init.yml"), path)
}

func TestExists(t *testing.T) {
	tests := map[string]struct {
		setup    func(t *testing.T, dir string)
		expected bool
	}{
		"file exists": {
			setup: func(t *testing.T, dir string) {
				t.Helper()
				initDir := filepath.Join(dir, ".autospec")
				require.NoError(t, os.MkdirAll(initDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(initDir, "init.yml"), []byte("version: 1.0.0"), 0o644))
			},
			expected: true,
		},
		"file does not exist": {
			setup:    func(t *testing.T, dir string) {},
			expected: false,
		},
		"directory exists but not file": {
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Join(dir, ".autospec"), 0o755))
			},
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			// Change to temp dir and test
			result := ExistsAt(filepath.Join(tmpDir, ".autospec", "init.yml"))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadFrom(t *testing.T) {
	tests := map[string]struct {
		content     string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, s *Settings)
	}{
		"valid YAML with all fields": {
			content: `version: "1.0.0"
settings_scope: global
autospec_version: "autospec v0.8.2"
agents:
  - name: claude
    configured: true
    settings_file: /home/user/.claude/settings.json
created_at: 2026-01-15T10:00:00Z
updated_at: 2026-01-15T11:00:00Z
`,
			wantErr: false,
			validate: func(t *testing.T, s *Settings) {
				t.Helper()
				assert.Equal(t, "1.0.0", s.Version)
				assert.Equal(t, "global", s.SettingsScope)
				assert.Equal(t, "autospec v0.8.2", s.AutospecVersion)
				require.Len(t, s.Agents, 1)
				assert.Equal(t, "claude", s.Agents[0].Name)
				assert.True(t, s.Agents[0].Configured)
				assert.Equal(t, "/home/user/.claude/settings.json", s.Agents[0].SettingsFile)
			},
		},
		"valid YAML with project scope": {
			content: `version: "1.0.0"
settings_scope: project
autospec_version: "autospec v0.8.2"
created_at: 2026-01-15T10:00:00Z
updated_at: 2026-01-15T10:00:00Z
`,
			wantErr: false,
			validate: func(t *testing.T, s *Settings) {
				t.Helper()
				assert.Equal(t, "project", s.SettingsScope)
				assert.Empty(t, s.Agents)
			},
		},
		"minimal valid YAML": {
			content: `version: "1.0.0"
settings_scope: global
`,
			wantErr: false,
			validate: func(t *testing.T, s *Settings) {
				t.Helper()
				assert.Equal(t, "1.0.0", s.Version)
				assert.Equal(t, "global", s.SettingsScope)
			},
		},
		"invalid YAML syntax": {
			content:     "version: [\n",
			wantErr:     true,
			errContains: "parsing init settings YAML",
		},
		"empty file": {
			content: "",
			wantErr: false,
			validate: func(t *testing.T, s *Settings) {
				t.Helper()
				assert.Empty(t, s.Version)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "init.yml")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o644))

			settings, err := LoadFrom(path)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, settings)
			}
		})
	}
}

func TestLoadFrom_MissingFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.yml")

	_, err := LoadFrom(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading init settings file")
}

func TestSaveTo(t *testing.T) {
	tests := map[string]struct {
		settings    *Settings
		existingDir bool
		validate    func(t *testing.T, path string)
	}{
		"save new file creates directory": {
			settings: &Settings{
				Version:         "1.0.0",
				SettingsScope:   ScopeGlobal,
				AutospecVersion: "autospec v0.8.2",
				Agents: []AgentEntry{
					{Name: "claude", Configured: true, SettingsFile: "/path/to/settings"},
				},
				CreatedAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 1, 15, 11, 0, 0, 0, time.UTC),
			},
			existingDir: false,
			validate: func(t *testing.T, path string) {
				t.Helper()
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Contains(t, string(data), "version: 1.0.0")
				assert.Contains(t, string(data), "settings_scope: global")
				assert.Contains(t, string(data), "autospec_version: autospec v0.8.2")
				assert.Contains(t, string(data), "name: claude")
			},
		},
		"save to existing directory": {
			settings: &Settings{
				Version:       "1.0.0",
				SettingsScope: ScopeProject,
			},
			existingDir: true,
			validate: func(t *testing.T, path string) {
				t.Helper()
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Contains(t, string(data), "settings_scope: project")
			},
		},
		"save overwrites existing file": {
			settings: &Settings{
				Version:       "2.0.0",
				SettingsScope: ScopeGlobal,
			},
			existingDir: true,
			validate: func(t *testing.T, path string) {
				t.Helper()
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Contains(t, string(data), "version: 2.0.0")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			subDir := filepath.Join(tmpDir, ".autospec")
			path := filepath.Join(subDir, "init.yml")

			if tt.existingDir {
				require.NoError(t, os.MkdirAll(subDir, 0o755))
			}

			err := tt.settings.SaveTo(path)
			require.NoError(t, err)

			// Verify file exists
			assert.FileExists(t, path)

			if tt.validate != nil {
				tt.validate(t, path)
			}
		})
	}
}

func TestSaveTo_RoundTrip(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "init.yml")

	original := &Settings{
		Version:         "1.0.0",
		SettingsScope:   ScopeGlobal,
		AutospecVersion: "autospec v0.8.2",
		Agents: []AgentEntry{
			{Name: "claude", Configured: true, SettingsFile: "/path/claude"},
			{Name: "opencode", Configured: false, SettingsFile: ""},
		},
		CreatedAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 15, 11, 30, 0, 0, time.UTC),
	}

	// Save
	err := original.SaveTo(path)
	require.NoError(t, err)

	// Load
	loaded, err := LoadFrom(path)
	require.NoError(t, err)

	// Verify round-trip
	assert.Equal(t, original.Version, loaded.Version)
	assert.Equal(t, original.SettingsScope, loaded.SettingsScope)
	assert.Equal(t, original.AutospecVersion, loaded.AutospecVersion)
	require.Len(t, loaded.Agents, 2)
	assert.Equal(t, original.Agents[0].Name, loaded.Agents[0].Name)
	assert.Equal(t, original.Agents[0].Configured, loaded.Agents[0].Configured)
	assert.Equal(t, original.Agents[1].Name, loaded.Agents[1].Name)
	assert.Equal(t, original.Agents[1].Configured, loaded.Agents[1].Configured)
}

func TestNewSettings(t *testing.T) {
	t.Parallel()

	before := time.Now()
	s := NewSettings("autospec v0.8.2")
	after := time.Now()

	assert.Equal(t, SchemaVersion, s.Version)
	assert.Equal(t, "autospec v0.8.2", s.AutospecVersion)
	assert.Empty(t, s.SettingsScope) // Must be set by caller
	assert.Empty(t, s.Agents)        // Must be set by caller
	assert.True(t, s.CreatedAt.After(before) || s.CreatedAt.Equal(before))
	assert.True(t, s.CreatedAt.Before(after) || s.CreatedAt.Equal(after))
	assert.Equal(t, s.CreatedAt, s.UpdatedAt)
}

func TestIsValidScope(t *testing.T) {
	tests := map[string]struct {
		scope    string
		expected bool
	}{
		"global is valid":       {scope: ScopeGlobal, expected: true},
		"project is valid":      {scope: ScopeProject, expected: true},
		"empty is invalid":      {scope: "", expected: false},
		"unknown is invalid":    {scope: "unknown", expected: false},
		"GLOBAL is invalid":     {scope: "GLOBAL", expected: false},
		"whitespace is invalid": {scope: " global ", expected: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := IsValidScope(tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaVersionConstant(t *testing.T) {
	t.Parallel()
	// Ensure schema version is set correctly
	assert.Equal(t, "1.0.0", SchemaVersion)
}

func TestScopeConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "global", ScopeGlobal)
	assert.Equal(t, "project", ScopeProject)
}
