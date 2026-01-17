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

func TestGetCacheLogBase(t *testing.T) {
	// Use a temp directory for testing
	tempDir := t.TempDir()

	originalXDG := os.Getenv("XDG_CACHE_HOME")
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)
	os.Setenv("XDG_CACHE_HOME", tempDir)

	tests := map[string]struct {
		cfg      *DAGExecutionConfig
		expected string
	}{
		"nil config uses XDG cache": {
			cfg:      nil,
			expected: filepath.Join(tempDir, "autospec", "dag-logs"),
		},
		"empty config uses XDG cache": {
			cfg:      &DAGExecutionConfig{},
			expected: filepath.Join(tempDir, "autospec", "dag-logs"),
		},
		"config with LogDir uses override": {
			cfg:      &DAGExecutionConfig{LogDir: "/custom/logs"},
			expected: "/custom/logs",
		},
		"config with empty LogDir uses XDG cache": {
			cfg:      &DAGExecutionConfig{LogDir: ""},
			expected: filepath.Join(tempDir, "autospec", "dag-logs"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetCacheLogBase(tt.cfg)
			if result != tt.expected {
				t.Errorf("GetCacheLogBase() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetCacheLogDirWithConfig(t *testing.T) {
	tests := map[string]struct {
		cfg       *DAGExecutionConfig
		projectID string
		dagID     string
		expected  string
	}{
		"nil config uses default path": {
			cfg:       nil,
			projectID: "test-project",
			dagID:     "test-dag",
			expected:  filepath.Join("autospec", "dag-logs", "test-project", "test-dag"),
		},
		"config override uses custom path": {
			cfg:       &DAGExecutionConfig{LogDir: "/custom/logs"},
			projectID: "test-project",
			dagID:     "test-dag",
			expected:  filepath.Join("/custom/logs", "test-project", "test-dag"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetCacheLogDirWithConfig(tt.cfg, tt.projectID, tt.dagID)
			if !strings.HasSuffix(result, tt.expected) {
				t.Errorf("GetCacheLogDirWithConfig() = %q, should end with %q", result, tt.expected)
			}
		})
	}
}

func TestGetLogFilePathWithConfig(t *testing.T) {
	tests := map[string]struct {
		cfg       *DAGExecutionConfig
		projectID string
		dagID     string
		specID    string
	}{
		"nil config uses default path": {
			cfg:       nil,
			projectID: "test-project",
			dagID:     "test-dag",
			specID:    "spec-001",
		},
		"config override uses custom path": {
			cfg:       &DAGExecutionConfig{LogDir: "/custom/logs"},
			projectID: "test-project",
			dagID:     "test-dag",
			specID:    "spec-001",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetLogFilePathWithConfig(tt.cfg, tt.projectID, tt.dagID, tt.specID)

			// Should end with specID.log
			expectedSuffix := tt.specID + ".log"
			if !strings.HasSuffix(result, expectedSuffix) {
				t.Errorf("GetLogFilePathWithConfig() = %q, should end with %q", result, expectedSuffix)
			}

			// Should contain the dag log directory structure
			dagLogDir := GetCacheLogDirWithConfig(tt.cfg, tt.projectID, tt.dagID)
			if !strings.HasPrefix(result, dagLogDir) {
				t.Errorf("GetLogFilePathWithConfig() = %q, should start with %q", result, dagLogDir)
			}
		})
	}
}

func TestEnsureCacheLogDirWithConfig(t *testing.T) {
	tempDir := t.TempDir()

	tests := map[string]struct {
		cfg       *DAGExecutionConfig
		projectID string
		dagID     string
	}{
		"nil config": {
			cfg:       nil,
			projectID: "nil-config-project",
			dagID:     "dag-1",
		},
		"custom log dir": {
			cfg:       &DAGExecutionConfig{LogDir: filepath.Join(tempDir, "custom-logs")},
			projectID: "custom-project",
			dagID:     "dag-2",
		},
	}

	// Set XDG_CACHE_HOME to temp directory for nil config case
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)
	os.Setenv("XDG_CACHE_HOME", tempDir)

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := EnsureCacheLogDirWithConfig(tt.cfg, tt.projectID, tt.dagID)
			if err != nil {
				t.Fatalf("EnsureCacheLogDirWithConfig() error = %v", err)
			}

			// Verify directory was created
			expectedDir := GetCacheLogDirWithConfig(tt.cfg, tt.projectID, tt.dagID)
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

func TestFormatBytes(t *testing.T) {
	tests := map[string]struct {
		bytes    int64
		expected string
	}{
		"zero bytes": {
			bytes:    0,
			expected: "0B",
		},
		"small bytes": {
			bytes:    512,
			expected: "512B",
		},
		"exactly 1 KB": {
			bytes:    1024,
			expected: "1KB",
		},
		"1.5 KB": {
			bytes:    1536,
			expected: "1.5KB",
		},
		"10 KB": {
			bytes:    10240,
			expected: "10KB",
		},
		"exactly 1 MB": {
			bytes:    1024 * 1024,
			expected: "1MB",
		},
		"50 MB": {
			bytes:    50 * 1024 * 1024,
			expected: "50MB",
		},
		"127 MB": {
			bytes:    127 * 1024 * 1024,
			expected: "127MB",
		},
		"exactly 1 GB": {
			bytes:    1024 * 1024 * 1024,
			expected: "1GB",
		},
		"2.5 GB": {
			bytes:    int64(2.5 * 1024 * 1024 * 1024),
			expected: "2.5GB",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestCalculateLogDirSize(t *testing.T) {
	tests := map[string]struct {
		setupFunc    func(dir string) error
		expectedSize int64
		expectFormat string
	}{
		"empty directory": {
			setupFunc:    func(_ string) error { return nil },
			expectedSize: 0,
			expectFormat: "0B",
		},
		"single small file": {
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "test.log"), make([]byte, 100), 0o644)
			},
			expectedSize: 100,
			expectFormat: "100B",
		},
		"multiple files": {
			setupFunc: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "a.log"), make([]byte, 1024), 0o644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "b.log"), make([]byte, 2048), 0o644)
			},
			expectedSize: 3072,
			expectFormat: "3KB",
		},
		"ignores subdirectories": {
			setupFunc: func(dir string) error {
				if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0o755); err != nil {
					return err
				}
				// File in subdir should not be counted
				if err := os.WriteFile(filepath.Join(dir, "subdir", "nested.log"), make([]byte, 1000), 0o644); err != nil {
					return err
				}
				// Only this file should be counted
				return os.WriteFile(filepath.Join(dir, "main.log"), make([]byte, 500), 0o644)
			},
			expectedSize: 500,
			expectFormat: "500B",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tempDir := t.TempDir()

			if err := tt.setupFunc(tempDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			size, formatted := CalculateLogDirSize(tempDir)

			if size != tt.expectedSize {
				t.Errorf("CalculateLogDirSize() size = %d, want %d", size, tt.expectedSize)
			}
			if formatted != tt.expectFormat {
				t.Errorf("CalculateLogDirSize() formatted = %q, want %q", formatted, tt.expectFormat)
			}
		})
	}
}

func TestCalculateLogDirSize_NonExistentDirectory(t *testing.T) {
	size, formatted := CalculateLogDirSize("/nonexistent/path/that/does/not/exist")

	if size != 0 {
		t.Errorf("CalculateLogDirSize(nonexistent) size = %d, want 0", size)
	}
	if formatted != "0B" {
		t.Errorf("CalculateLogDirSize(nonexistent) formatted = %q, want \"0B\"", formatted)
	}
}
