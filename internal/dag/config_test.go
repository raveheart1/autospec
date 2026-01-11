package dag

import (
	"os"
	"testing"

	"github.com/ariel-frischer/autospec/internal/worktree"
)

func TestDefaultDAGConfig(t *testing.T) {
	cfg := DefaultDAGConfig()

	if cfg.OnConflict != "manual" {
		t.Errorf("OnConflict: got %q, want %q", cfg.OnConflict, "manual")
	}
	if cfg.MaxSpecRetries != 0 {
		t.Errorf("MaxSpecRetries: got %d, want %d", cfg.MaxSpecRetries, 0)
	}
	if cfg.MaxLogSize != "50MB" {
		t.Errorf("MaxLogSize: got %q, want %q", cfg.MaxLogSize, "50MB")
	}
}

func TestLoadDAGConfig(t *testing.T) {
	tests := map[string]struct {
		input    *DAGExecutionConfig
		envVars  map[string]string
		expected *DAGExecutionConfig
	}{
		"nil config uses defaults": {
			input:   nil,
			envVars: nil,
			expected: &DAGExecutionConfig{
				OnConflict:     "manual",
				BaseBranch:     "",
				MaxSpecRetries: 0,
				MaxLogSize:     "50MB",
			},
		},
		"provided config overrides defaults": {
			input: &DAGExecutionConfig{
				OnConflict:     "agent",
				BaseBranch:     "main",
				MaxSpecRetries: 3,
				MaxLogSize:     "100MB",
			},
			envVars: nil,
			expected: &DAGExecutionConfig{
				OnConflict:     "agent",
				BaseBranch:     "main",
				MaxSpecRetries: 3,
				MaxLogSize:     "100MB",
			},
		},
		"env vars override provided config": {
			input: &DAGExecutionConfig{
				OnConflict:     "agent",
				BaseBranch:     "main",
				MaxSpecRetries: 3,
				MaxLogSize:     "100MB",
			},
			envVars: map[string]string{
				"AUTOSPEC_DAG_ON_CONFLICT":      "manual",
				"AUTOSPEC_DAG_BASE_BRANCH":      "develop",
				"AUTOSPEC_DAG_MAX_SPEC_RETRIES": "5",
				"AUTOSPEC_DAG_MAX_LOG_SIZE":     "200MB",
			},
			expected: &DAGExecutionConfig{
				OnConflict:     "manual",
				BaseBranch:     "develop",
				MaxSpecRetries: 5,
				MaxLogSize:     "200MB",
			},
		},
		"partial config with env overrides": {
			input: &DAGExecutionConfig{
				OnConflict: "agent",
			},
			envVars: map[string]string{
				"AUTOSPEC_DAG_MAX_SPEC_RETRIES": "2",
			},
			expected: &DAGExecutionConfig{
				OnConflict:     "agent",
				BaseBranch:     "",
				MaxSpecRetries: 2,
				MaxLogSize:     "50MB",
			},
		},
		"invalid env var ignored": {
			input: nil,
			envVars: map[string]string{
				"AUTOSPEC_DAG_MAX_SPEC_RETRIES": "not-a-number",
			},
			expected: &DAGExecutionConfig{
				OnConflict:     "manual",
				BaseBranch:     "",
				MaxSpecRetries: 0,
				MaxLogSize:     "50MB",
			},
		},
		"base branch from env only": {
			input: nil,
			envVars: map[string]string{
				"AUTOSPEC_DAG_BASE_BRANCH": "feature-branch",
			},
			expected: &DAGExecutionConfig{
				OnConflict:     "manual",
				BaseBranch:     "feature-branch",
				MaxSpecRetries: 0,
				MaxLogSize:     "50MB",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Clear all relevant env vars first
			os.Unsetenv("AUTOSPEC_DAG_ON_CONFLICT")
			os.Unsetenv("AUTOSPEC_DAG_BASE_BRANCH")
			os.Unsetenv("AUTOSPEC_DAG_MAX_SPEC_RETRIES")
			os.Unsetenv("AUTOSPEC_DAG_MAX_LOG_SIZE")

			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := LoadDAGConfig(tt.input)

			if result.OnConflict != tt.expected.OnConflict {
				t.Errorf("OnConflict: got %q, want %q", result.OnConflict, tt.expected.OnConflict)
			}
			if result.BaseBranch != tt.expected.BaseBranch {
				t.Errorf("BaseBranch: got %q, want %q", result.BaseBranch, tt.expected.BaseBranch)
			}
			if result.MaxSpecRetries != tt.expected.MaxSpecRetries {
				t.Errorf("MaxSpecRetries: got %d, want %d", result.MaxSpecRetries, tt.expected.MaxSpecRetries)
			}
			if result.MaxLogSize != tt.expected.MaxLogSize {
				t.Errorf("MaxLogSize: got %q, want %q", result.MaxLogSize, tt.expected.MaxLogSize)
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected int64
		wantErr  bool
	}{
		"50MB": {
			input:    "50MB",
			expected: 50 * 1024 * 1024,
			wantErr:  false,
		},
		"100MB": {
			input:    "100MB",
			expected: 100 * 1024 * 1024,
			wantErr:  false,
		},
		"1GB": {
			input:    "1GB",
			expected: 1 * 1024 * 1024 * 1024,
			wantErr:  false,
		},
		"512KB": {
			input:    "512KB",
			expected: 512 * 1024,
			wantErr:  false,
		},
		"1024B": {
			input:    "1024B",
			expected: 1024,
			wantErr:  false,
		},
		"lowercase mb": {
			input:    "50mb",
			expected: 50 * 1024 * 1024,
			wantErr:  false,
		},
		"with spaces": {
			input:    "  50 MB  ",
			expected: 50 * 1024 * 1024,
			wantErr:  false,
		},
		"empty string": {
			input:   "",
			wantErr: true,
		},
		"invalid format": {
			input:   "50",
			wantErr: true,
		},
		"invalid unit": {
			input:   "50XB",
			wantErr: true,
		},
		"negative": {
			input:   "-50MB",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := ParseSize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestMaxLogSizeBytes(t *testing.T) {
	tests := map[string]struct {
		maxLogSize string
		expected   int64
	}{
		"default 50MB": {
			maxLogSize: "",
			expected:   50 * 1024 * 1024,
		},
		"explicit 50MB": {
			maxLogSize: "50MB",
			expected:   50 * 1024 * 1024,
		},
		"100MB": {
			maxLogSize: "100MB",
			expected:   100 * 1024 * 1024,
		},
		"1GB": {
			maxLogSize: "1GB",
			expected:   1 * 1024 * 1024 * 1024,
		},
		"invalid falls back to default": {
			maxLogSize: "invalid",
			expected:   50 * 1024 * 1024,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &DAGExecutionConfig{MaxLogSize: tt.maxLogSize}
			result := cfg.MaxLogSizeBytes()
			if result != tt.expected {
				t.Errorf("got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestLoadWorktreeConfig(t *testing.T) {
	tests := map[string]struct {
		input    *worktree.WorktreeConfig
		envVars  map[string]string
		expected *worktree.WorktreeConfig
	}{
		"nil config uses defaults": {
			input:   nil,
			envVars: nil,
			expected: &worktree.WorktreeConfig{
				BaseDir:     "",
				Prefix:      "",
				SetupScript: "",
				AutoSetup:   true,
				TrackStatus: true,
				CopyDirs:    []string{".autospec", ".claude"},
			},
		},
		"provided config overrides defaults": {
			input: &worktree.WorktreeConfig{
				BaseDir:     "/custom/path",
				Prefix:      "wt-",
				SetupScript: "setup.sh",
				AutoSetup:   false,
				TrackStatus: false,
				CopyDirs:    []string{".custom"},
			},
			envVars: nil,
			expected: &worktree.WorktreeConfig{
				BaseDir:     "/custom/path",
				Prefix:      "wt-",
				SetupScript: "setup.sh",
				AutoSetup:   false,
				TrackStatus: false,
				CopyDirs:    []string{".custom"},
			},
		},
		"env vars override provided config": {
			input: &worktree.WorktreeConfig{
				BaseDir: "/custom/path",
				Prefix:  "wt-",
			},
			envVars: map[string]string{
				"AUTOSPEC_WORKTREE_BASE_DIR":     "/env/path",
				"AUTOSPEC_WORKTREE_PREFIX":       "env-",
				"AUTOSPEC_WORKTREE_SETUP_SCRIPT": "env-setup.sh",
				"AUTOSPEC_WORKTREE_AUTO_SETUP":   "false",
				"AUTOSPEC_WORKTREE_TRACK_STATUS": "false",
			},
			expected: &worktree.WorktreeConfig{
				BaseDir:     "/env/path",
				Prefix:      "env-",
				SetupScript: "env-setup.sh",
				AutoSetup:   false,
				TrackStatus: false,
				CopyDirs:    []string{".autospec", ".claude"},
			},
		},
		"bool env vars with 1 value": {
			input: nil,
			envVars: map[string]string{
				"AUTOSPEC_WORKTREE_AUTO_SETUP":   "1",
				"AUTOSPEC_WORKTREE_TRACK_STATUS": "1",
			},
			expected: &worktree.WorktreeConfig{
				BaseDir:     "",
				Prefix:      "",
				SetupScript: "",
				AutoSetup:   true,
				TrackStatus: true,
				CopyDirs:    []string{".autospec", ".claude"},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Clear all relevant env vars first
			os.Unsetenv("AUTOSPEC_WORKTREE_BASE_DIR")
			os.Unsetenv("AUTOSPEC_WORKTREE_PREFIX")
			os.Unsetenv("AUTOSPEC_WORKTREE_SETUP_SCRIPT")
			os.Unsetenv("AUTOSPEC_WORKTREE_AUTO_SETUP")
			os.Unsetenv("AUTOSPEC_WORKTREE_TRACK_STATUS")

			// Set test-specific environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := LoadWorktreeConfig(tt.input)

			if result.BaseDir != tt.expected.BaseDir {
				t.Errorf("BaseDir: got %q, want %q", result.BaseDir, tt.expected.BaseDir)
			}
			if result.Prefix != tt.expected.Prefix {
				t.Errorf("Prefix: got %q, want %q", result.Prefix, tt.expected.Prefix)
			}
			if result.SetupScript != tt.expected.SetupScript {
				t.Errorf("SetupScript: got %q, want %q", result.SetupScript, tt.expected.SetupScript)
			}
			if result.AutoSetup != tt.expected.AutoSetup {
				t.Errorf("AutoSetup: got %v, want %v", result.AutoSetup, tt.expected.AutoSetup)
			}
			if result.TrackStatus != tt.expected.TrackStatus {
				t.Errorf("TrackStatus: got %v, want %v", result.TrackStatus, tt.expected.TrackStatus)
			}
		})
	}
}
