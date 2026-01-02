// Package opencode_test tests OpenCode settings file management and permission validation.
// Related: /home/ari/repos/autospec/internal/opencode/settings.go
// Tags: opencode, settings, permissions, json, validation

package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup       func(t *testing.T, dir string)
		wantErr     bool
		wantErrMsg  string
		checkResult func(t *testing.T, s *Settings)
	}{
		"missing file returns empty settings": {
			setup: func(t *testing.T, dir string) {
				// No setup - file doesn't exist
			},
			checkResult: func(t *testing.T, s *Settings) {
				assert.NotNil(t, s)
				assert.False(t, s.Exists())
				assert.Empty(t, s.Permission.Bash)
			},
		},
		"empty file returns empty settings": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, "")
			},
			checkResult: func(t *testing.T, s *Settings) {
				assert.NotNil(t, s)
				assert.True(t, s.Exists())
				assert.Empty(t, s.Permission.Bash)
			},
		},
		"valid JSON with permissions": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{"permission": {"bash": {"autospec *": "allow"}}}`)
			},
			checkResult: func(t *testing.T, s *Settings) {
				assert.NotNil(t, s)
				assert.True(t, s.Exists())
				assert.Equal(t, "allow", s.Permission.Bash["autospec *"])
			},
		},
		"malformed JSON returns error": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{invalid json}`)
			},
			wantErr:    true,
			wantErrMsg: "parsing settings file",
		},
		"preserves extra fields": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{
					"permission": {"bash": {"autospec *": "allow"}},
					"model": "claude-3-opus",
					"custom_field": "value"
				}`)
			},
			checkResult: func(t *testing.T, s *Settings) {
				assert.NotNil(t, s)
				assert.Contains(t, s.extra, "model")
				assert.Contains(t, s.extra, "custom_field")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, dir)
			}

			s, err := Load(dir)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
				return
			}

			require.NoError(t, err)
			if tt.checkResult != nil {
				tt.checkResult(t, s)
			}
		})
	}
}

func TestCheckBashPermission(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		bashPerms map[string]string
		pattern   string
		want      string
	}{
		"empty permissions": {
			bashPerms: nil,
			pattern:   "autospec *",
			want:      "",
		},
		"pattern not configured": {
			bashPerms: map[string]string{"git *": "allow"},
			pattern:   "autospec *",
			want:      "",
		},
		"pattern allowed": {
			bashPerms: map[string]string{"autospec *": "allow"},
			pattern:   "autospec *",
			want:      "allow",
		},
		"pattern denied": {
			bashPerms: map[string]string{"autospec *": "deny"},
			pattern:   "autospec *",
			want:      "deny",
		},
		"pattern asks": {
			bashPerms: map[string]string{"autospec *": "ask"},
			pattern:   "autospec *",
			want:      "ask",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := &Settings{
				Permission: Permission{Bash: tt.bashPerms},
			}

			got := s.CheckBashPermission(tt.pattern)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAddBashPermission(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialPerms map[string]string
		pattern      string
		level        string
		wantPerms    map[string]string
	}{
		"add to empty permissions": {
			initialPerms: nil,
			pattern:      "autospec *",
			level:        "allow",
			wantPerms:    map[string]string{"autospec *": "allow"},
		},
		"add new pattern": {
			initialPerms: map[string]string{"git *": "allow"},
			pattern:      "autospec *",
			level:        "allow",
			wantPerms:    map[string]string{"git *": "allow", "autospec *": "allow"},
		},
		"update existing pattern": {
			initialPerms: map[string]string{"autospec *": "ask"},
			pattern:      "autospec *",
			level:        "allow",
			wantPerms:    map[string]string{"autospec *": "allow"},
		},
		"idempotent add": {
			initialPerms: map[string]string{"autospec *": "allow"},
			pattern:      "autospec *",
			level:        "allow",
			wantPerms:    map[string]string{"autospec *": "allow"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := &Settings{
				Permission: Permission{Bash: tt.initialPerms},
			}

			s.AddBashPermission(tt.pattern, tt.level)

			assert.Equal(t, tt.wantPerms, s.Permission.Bash)
		})
	}
}

func TestHasRequiredPermission(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		bashPerms map[string]string
		want      bool
	}{
		"no permissions": {
			bashPerms: nil,
			want:      false,
		},
		"wrong pattern": {
			bashPerms: map[string]string{"git *": "allow"},
			want:      false,
		},
		"pattern denied": {
			bashPerms: map[string]string{"autospec *": "deny"},
			want:      false,
		},
		"pattern asks": {
			bashPerms: map[string]string{"autospec *": "ask"},
			want:      false,
		},
		"pattern allowed": {
			bashPerms: map[string]string{"autospec *": "allow"},
			want:      true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := &Settings{
				Permission: Permission{Bash: tt.bashPerms},
			}

			got := s.HasRequiredPermission()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsPermissionDenied(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		bashPerms map[string]string
		want      bool
	}{
		"no permissions": {
			bashPerms: nil,
			want:      false,
		},
		"pattern allowed": {
			bashPerms: map[string]string{"autospec *": "allow"},
			want:      false,
		},
		"pattern asks": {
			bashPerms: map[string]string{"autospec *": "ask"},
			want:      false,
		},
		"pattern denied": {
			bashPerms: map[string]string{"autospec *": "deny"},
			want:      true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := &Settings{
				Permission: Permission{Bash: tt.bashPerms},
			}

			got := s.IsPermissionDenied()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheck(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup           func(t *testing.T, dir string)
		wantStatus      SettingsStatus
		wantMsgContains string
	}{
		"missing file": {
			setup: func(t *testing.T, dir string) {
				// No setup - file doesn't exist
			},
			wantStatus:      StatusMissing,
			wantMsgContains: "not found",
		},
		"permission denied": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{
					"permission": {"bash": {"autospec *": "deny"}}
				}`)
			},
			wantStatus:      StatusDenied,
			wantMsgContains: "explicitly denied",
		},
		"permission missing": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{
					"permission": {"bash": {"git *": "allow"}}
				}`)
			},
			wantStatus:      StatusNeedsPermission,
			wantMsgContains: "missing",
		},
		"permission asks only": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{
					"permission": {"bash": {"autospec *": "ask"}}
				}`)
			},
			wantStatus:      StatusNeedsPermission,
			wantMsgContains: "missing",
		},
		"permission configured": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{
					"permission": {"bash": {"autospec *": "allow"}}
				}`)
			},
			wantStatus:      StatusConfigured,
			wantMsgContains: "configured",
		},
		"empty bash map": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{
					"permission": {"bash": {}}
				}`)
			},
			wantStatus:      StatusNeedsPermission,
			wantMsgContains: "missing",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, dir)
			}

			s, err := Load(dir)
			require.NoError(t, err)

			result := s.Check()

			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Contains(t, result.Message, tt.wantMsgContains)
		})
	}
}

func TestSave(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup       func(t *testing.T, s *Settings)
		checkResult func(t *testing.T, dir string, data []byte)
	}{
		"creates file": {
			setup: func(t *testing.T, s *Settings) {
				s.AddBashPermission("autospec *", "allow")
			},
			checkResult: func(t *testing.T, dir string, data []byte) {
				assert.FileExists(t, filepath.Join(dir, SettingsFileName))
			},
		},
		"writes pretty-printed JSON": {
			setup: func(t *testing.T, s *Settings) {
				s.AddBashPermission("autospec *", "allow")
			},
			checkResult: func(t *testing.T, dir string, data []byte) {
				assert.Contains(t, string(data), "  ") // Indentation
				assert.True(t, json.Valid(data))
			},
		},
		"preserves existing fields": {
			setup: func(t *testing.T, s *Settings) {
				s.extra["model"] = json.RawMessage(`"claude-3-opus"`)
				s.extra["custom"] = json.RawMessage(`{"nested": "value"}`)
				s.AddBashPermission("autospec *", "allow")
			},
			checkResult: func(t *testing.T, dir string, data []byte) {
				assert.Contains(t, string(data), "model")
				assert.Contains(t, string(data), "custom")
				assert.Contains(t, string(data), "autospec *")
			},
		},
		"ends with newline": {
			setup: func(t *testing.T, s *Settings) {
				s.AddBashPermission("autospec *", "allow")
			},
			checkResult: func(t *testing.T, dir string, data []byte) {
				assert.True(t, len(data) > 0)
				assert.Equal(t, byte('\n'), data[len(data)-1])
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			s := NewSettings(dir)

			if tt.setup != nil {
				tt.setup(t, s)
			}

			err := s.Save()
			require.NoError(t, err)

			data, err := os.ReadFile(s.FilePath())
			require.NoError(t, err)

			if tt.checkResult != nil {
				tt.checkResult(t, dir, data)
			}
		})
	}
}

func TestSaveAtomicWrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := NewSettings(dir)
	s.AddBashPermission("autospec *", "allow")

	// Save the file
	err := s.Save()
	require.NoError(t, err)

	// Verify no temp files left behind
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, entry := range entries {
		assert.False(t,
			filepath.Ext(entry.Name()) == ".tmp",
			"temp file should not remain: %s", entry.Name())
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	// Test that loading, modifying, and saving preserves all fields
	dir := t.TempDir()

	original := `{
  "permission": {
    "bash": {
      "git *": "allow",
      "npm *": "ask"
    }
  },
  "model": "claude-3-opus",
  "agent": {
    "maxIterations": 50
  }
}`
	createSettingsFile(t, dir, original)

	// Load
	s, err := Load(dir)
	require.NoError(t, err)

	// Modify
	s.AddBashPermission("autospec *", "allow")

	// Save
	err = s.Save()
	require.NoError(t, err)

	// Reload and verify
	s2, err := Load(dir)
	require.NoError(t, err)

	// Check all permissions preserved
	assert.Equal(t, "allow", s2.CheckBashPermission("git *"))
	assert.Equal(t, "ask", s2.CheckBashPermission("npm *"))
	assert.Equal(t, "allow", s2.CheckBashPermission("autospec *"))

	// Check extra fields preserved
	assert.Contains(t, s2.extra, "model")
	assert.Contains(t, s2.extra, "agent")
}

func TestSettingsStatus_String(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status SettingsStatus
		want   string
	}{
		"Configured":      {status: StatusConfigured, want: "Configured"},
		"Missing":         {status: StatusMissing, want: "Missing"},
		"NeedsPermission": {status: StatusNeedsPermission, want: "NeedsPermission"},
		"Denied":          {status: StatusDenied, want: "Denied"},
		"Unknown":         {status: SettingsStatus(99), want: "Unknown"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestCheckInDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createSettingsFile(t, dir, `{"permission": {"bash": {"autospec *": "allow"}}}`)

	result, err := CheckInDir(dir)

	require.NoError(t, err)
	assert.Equal(t, StatusConfigured, result.Status)
}

func TestNewSettings(t *testing.T) {
	t.Parallel()

	dir := "/some/project/dir"
	s := NewSettings(dir)

	assert.NotNil(t, s)
	assert.Equal(t, filepath.Join(dir, SettingsFileName), s.FilePath())
	assert.NotNil(t, s.Permission.Bash)
	assert.NotNil(t, s.extra)
}

func TestFilePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := NewSettings(dir)

	expected := filepath.Join(dir, SettingsFileName)
	assert.Equal(t, expected, s.FilePath())
}

func TestExists(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup func(t *testing.T, dir string)
		want  bool
	}{
		"file does not exist": {
			setup: func(t *testing.T, dir string) {},
			want:  false,
		},
		"file exists": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{}`)
			},
			want: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)

			s, err := Load(dir)
			require.NoError(t, err)

			assert.Equal(t, tt.want, s.Exists())
		})
	}
}

func TestAddBashPermissionIdempotency(t *testing.T) {
	t.Parallel()

	// Calling AddBashPermission multiple times with same values should be safe
	s := NewSettings(t.TempDir())

	s.AddBashPermission("autospec *", "allow")
	s.AddBashPermission("autospec *", "allow")
	s.AddBashPermission("autospec *", "allow")

	assert.Equal(t, map[string]string{"autospec *": "allow"}, s.Permission.Bash)
}

// Helper functions

func createSettingsFile(t *testing.T, dir, content string) {
	t.Helper()
	settingsPath := filepath.Join(dir, SettingsFileName)
	err := os.WriteFile(settingsPath, []byte(content), 0o644)
	require.NoError(t, err)
}
