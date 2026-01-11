package dag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetCacheBase(t *testing.T) {
	tests := map[string]struct {
		xdgCacheHome string
		expectXDG    bool
	}{
		"with XDG_CACHE_HOME set": {
			xdgCacheHome: "/custom/cache",
			expectXDG:    true,
		},
		"without XDG_CACHE_HOME": {
			xdgCacheHome: "",
			expectXDG:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Save and restore original env
			originalXDG := os.Getenv("XDG_CACHE_HOME")
			defer os.Setenv("XDG_CACHE_HOME", originalXDG)

			// Set test environment
			if tt.xdgCacheHome != "" {
				os.Setenv("XDG_CACHE_HOME", tt.xdgCacheHome)
			} else {
				os.Unsetenv("XDG_CACHE_HOME")
			}

			result := GetCacheBase()

			if tt.expectXDG {
				if result != tt.xdgCacheHome {
					t.Errorf("GetCacheBase() = %q, want %q", result, tt.xdgCacheHome)
				}
			} else {
				// Without XDG_CACHE_HOME, should return os.UserCacheDir() result
				if result == "" {
					t.Error("GetCacheBase() returned empty string")
				}
				// Should be an absolute path or relative fallback
				if !filepath.IsAbs(result) && !strings.HasPrefix(result, ".") {
					t.Errorf("GetCacheBase() = %q, expected absolute path or relative fallback", result)
				}
			}
		})
	}
}

func TestGetCacheLogDir(t *testing.T) {
	tests := map[string]struct {
		projectID string
		dagID     string
	}{
		"simple project and dag": {
			projectID: "github-com-user-repo",
			dagID:     "my-dag",
		},
		"with hyphens and numbers": {
			projectID: "gitlab-com-org-project-123",
			dagID:     "dag-2025-01",
		},
		"short ids": {
			projectID: "abc123",
			dagID:     "x",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetCacheLogDir(tt.projectID, tt.dagID)

			// Should contain autospec/dag-logs path structure
			if !strings.Contains(result, filepath.Join("autospec", "dag-logs")) {
				t.Errorf("GetCacheLogDir() = %q, missing autospec/dag-logs", result)
			}

			// Should end with projectID/dagID
			expectedSuffix := filepath.Join(tt.projectID, tt.dagID)
			if !strings.HasSuffix(result, expectedSuffix) {
				t.Errorf("GetCacheLogDir() = %q, should end with %q", result, expectedSuffix)
			}
		})
	}
}

func TestGetLogFilePath(t *testing.T) {
	tests := map[string]struct {
		projectID string
		dagID     string
		specID    string
	}{
		"typical spec": {
			projectID: "github-com-user-repo",
			dagID:     "my-dag",
			specID:    "spec-001",
		},
		"spec with numbers": {
			projectID: "project-a",
			dagID:     "dag-1",
			specID:    "feature-123-auth",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetLogFilePath(tt.projectID, tt.dagID, tt.specID)

			// Should end with specID.log
			expectedSuffix := tt.specID + ".log"
			if !strings.HasSuffix(result, expectedSuffix) {
				t.Errorf("GetLogFilePath() = %q, should end with %q", result, expectedSuffix)
			}

			// Should contain the dag log directory structure
			dagLogDir := GetCacheLogDir(tt.projectID, tt.dagID)
			if !strings.HasPrefix(result, dagLogDir) {
				t.Errorf("GetLogFilePath() = %q, should start with %q", result, dagLogDir)
			}
		})
	}
}

func TestEnsureCacheLogDir(t *testing.T) {
	// Use a temp directory for testing
	tempDir := t.TempDir()

	// Set XDG_CACHE_HOME to temp directory
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)
	os.Setenv("XDG_CACHE_HOME", tempDir)

	tests := map[string]struct {
		projectID string
		dagID     string
	}{
		"create new directory": {
			projectID: "test-project",
			dagID:     "test-dag",
		},
		"with nested path": {
			projectID: "github-com-user-repo",
			dagID:     "complex-dag-123",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := EnsureCacheLogDir(tt.projectID, tt.dagID)
			if err != nil {
				t.Fatalf("EnsureCacheLogDir() error = %v", err)
			}

			// Verify directory was created
			expectedDir := GetCacheLogDir(tt.projectID, tt.dagID)
			info, err := os.Stat(expectedDir)
			if err != nil {
				t.Fatalf("Directory not created: %v", err)
			}
			if !info.IsDir() {
				t.Error("Created path is not a directory")
			}
		})
	}
}

func TestEnsureCacheLogDir_Idempotent(t *testing.T) {
	// Use a temp directory for testing
	tempDir := t.TempDir()

	originalXDG := os.Getenv("XDG_CACHE_HOME")
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)
	os.Setenv("XDG_CACHE_HOME", tempDir)

	projectID := "idempotent-test"
	dagID := "dag-1"

	// Call twice - should not error on second call
	if err := EnsureCacheLogDir(projectID, dagID); err != nil {
		t.Fatalf("First EnsureCacheLogDir() error = %v", err)
	}

	if err := EnsureCacheLogDir(projectID, dagID); err != nil {
		t.Fatalf("Second EnsureCacheLogDir() error = %v", err)
	}
}
