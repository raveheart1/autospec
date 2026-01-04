package config

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	// Note: Cannot use t.Parallel() because ResolvePath calls os.Getwd()

	// Get current working directory for test assertions
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Get user home directory for tilde tests
	u, err := user.Current()
	if err != nil {
		t.Fatalf("failed to get current user: %v", err)
	}
	homeDir := u.HomeDir

	tests := map[string]struct {
		input       string
		expected    string
		expectError bool
	}{
		"empty path returns cwd": {
			input:    "",
			expected: cwd,
		},
		"dot returns cwd": {
			input:    ".",
			expected: cwd,
		},
		"tilde alone returns home": {
			input:    "~",
			expected: homeDir,
		},
		"tilde with path expands": {
			input:    "~/projects",
			expected: filepath.Join(homeDir, "projects"),
		},
		"tilde nested path expands": {
			input:    "~/foo/bar/baz",
			expected: filepath.Join(homeDir, "foo", "bar", "baz"),
		},
		"relative path resolves against cwd": {
			input:    "my-project",
			expected: filepath.Join(cwd, "my-project"),
		},
		"relative nested path resolves": {
			input:    "foo/bar/baz",
			expected: filepath.Join(cwd, "foo", "bar", "baz"),
		},
		"absolute path unchanged": {
			input:    "/tmp/test-project",
			expected: "/tmp/test-project",
		},
		"absolute nested path unchanged": {
			input:    "/home/user/projects/test",
			expected: "/home/user/projects/test",
		},
		"path with spaces": {
			input:    "my project",
			expected: filepath.Join(cwd, "my project"),
		},
		"path with special characters": {
			input:    "project-v2.0_final",
			expected: filepath.Join(cwd, "project-v2.0_final"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := ResolvePath(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExpandTilde(t *testing.T) {
	t.Parallel()

	u, err := user.Current()
	if err != nil {
		t.Fatalf("failed to get current user: %v", err)
	}
	homeDir := u.HomeDir

	tests := map[string]struct {
		input    string
		expected string
	}{
		"tilde alone": {
			input:    "~",
			expected: homeDir,
		},
		"tilde with slash": {
			input:    "~/",
			expected: homeDir,
		},
		"tilde with path": {
			input:    "~/projects",
			expected: filepath.Join(homeDir, "projects"),
		},
		"tilde nested": {
			input:    "~/a/b/c",
			expected: filepath.Join(homeDir, "a", "b", "c"),
		},
		"no tilde passthrough": {
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		"relative passthrough": {
			input:    "relative/path",
			expected: "relative/path",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := expandTilde(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEnsureDirectory(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup       func(t *testing.T, tmpDir string) string
		expectError bool
		errorMsg    string
	}{
		"creates non-existent directory": {
			setup: func(t *testing.T, tmpDir string) string {
				return filepath.Join(tmpDir, "new-dir")
			},
			expectError: false,
		},
		"creates nested non-existent directories": {
			setup: func(t *testing.T, tmpDir string) string {
				return filepath.Join(tmpDir, "a", "b", "c")
			},
			expectError: false,
		},
		"succeeds if directory already exists": {
			setup: func(t *testing.T, tmpDir string) string {
				dir := filepath.Join(tmpDir, "existing")
				if err := os.Mkdir(dir, 0o755); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return dir
			},
			expectError: false,
		},
		"fails if path is a file": {
			setup: func(t *testing.T, tmpDir string) string {
				filePath := filepath.Join(tmpDir, "somefile")
				if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return filePath
			},
			expectError: true,
			errorMsg:    "path exists and is not a directory",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create isolated temp directory for this test
			tmpDir := t.TempDir()
			targetPath := tt.setup(t, tmpDir)

			err := EnsureDirectory(targetPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" {
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("error %q does not contain %q", err.Error(), tt.errorMsg)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify directory was created
			info, err := os.Stat(targetPath)
			if err != nil {
				t.Errorf("directory was not created: %v", err)
				return
			}
			if !info.IsDir() {
				t.Errorf("path is not a directory")
			}
		})
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestResolveTargetDirectory(t *testing.T) {
	// Note: Cannot use t.Parallel() because resolveTargetDirectory calls os.Getwd()
	// internally, and other tests in this package change the working directory.

	// Get current working directory for comparisons
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	tests := map[string]struct {
		args        []string
		here        bool
		expected    string
		expectError bool
	}{
		"no args, no flag returns empty": {
			args:     []string{},
			here:     false,
			expected: "",
		},
		"empty args slice returns empty": {
			args:     nil,
			here:     false,
			expected: "",
		},
		"dot returns empty (same as cwd)": {
			args:     []string{"."},
			here:     false,
			expected: "",
		},
		"here flag with no args returns empty": {
			args:     []string{},
			here:     true,
			expected: "",
		},
		"relative path is resolved": {
			args:     []string{"test-project"},
			here:     false,
			expected: filepath.Join(cwd, "test-project"),
		},
		"absolute path is unchanged": {
			args:     []string{"/tmp/test-init"},
			here:     false,
			expected: "/tmp/test-init",
		},
		"path arg overrides here flag": {
			args:     []string{"/tmp/other-project"},
			here:     true,
			expected: "/tmp/other-project",
		},
		"nested relative path": {
			args:     []string{"a/b/c"},
			here:     false,
			expected: filepath.Join(cwd, "a/b/c"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := resolveTargetDirectory(tt.args, tt.here)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}
