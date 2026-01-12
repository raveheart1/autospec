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
	if cfg.Autocommit == nil || !*cfg.Autocommit {
		t.Errorf("Autocommit: expected true by default")
	}
	if cfg.AutocommitRetries != 1 {
		t.Errorf("AutocommitRetries: got %d, want %d", cfg.AutocommitRetries, 1)
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
				LogDir:         "",
			},
		},
		"provided config overrides defaults": {
			input: &DAGExecutionConfig{
				OnConflict:     "agent",
				BaseBranch:     "main",
				MaxSpecRetries: 3,
				MaxLogSize:     "100MB",
				LogDir:         "/custom/logs",
			},
			envVars: nil,
			expected: &DAGExecutionConfig{
				OnConflict:     "agent",
				BaseBranch:     "main",
				MaxSpecRetries: 3,
				MaxLogSize:     "100MB",
				LogDir:         "/custom/logs",
			},
		},
		"env vars override provided config": {
			input: &DAGExecutionConfig{
				OnConflict:     "agent",
				BaseBranch:     "main",
				MaxSpecRetries: 3,
				MaxLogSize:     "100MB",
				LogDir:         "/custom/logs",
			},
			envVars: map[string]string{
				"AUTOSPEC_DAG_ON_CONFLICT":      "manual",
				"AUTOSPEC_DAG_BASE_BRANCH":      "develop",
				"AUTOSPEC_DAG_MAX_SPEC_RETRIES": "5",
				"AUTOSPEC_DAG_MAX_LOG_SIZE":     "200MB",
				"AUTOSPEC_DAG_LOG_DIR":          "/env/logs",
			},
			expected: &DAGExecutionConfig{
				OnConflict:     "manual",
				BaseBranch:     "develop",
				MaxSpecRetries: 5,
				MaxLogSize:     "200MB",
				LogDir:         "/env/logs",
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
				LogDir:         "",
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
				LogDir:         "",
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
				LogDir:         "",
			},
		},
		"log dir from config only": {
			input: &DAGExecutionConfig{
				LogDir: "/custom/log/path",
			},
			envVars: nil,
			expected: &DAGExecutionConfig{
				OnConflict:     "manual",
				BaseBranch:     "",
				MaxSpecRetries: 0,
				MaxLogSize:     "50MB",
				LogDir:         "/custom/log/path",
			},
		},
		"log dir from env only": {
			input: nil,
			envVars: map[string]string{
				"AUTOSPEC_DAG_LOG_DIR": "/env/log/path",
			},
			expected: &DAGExecutionConfig{
				OnConflict:     "manual",
				BaseBranch:     "",
				MaxSpecRetries: 0,
				MaxLogSize:     "50MB",
				LogDir:         "/env/log/path",
			},
		},
		"log dir env overrides config": {
			input: &DAGExecutionConfig{
				LogDir: "/config/logs",
			},
			envVars: map[string]string{
				"AUTOSPEC_DAG_LOG_DIR": "/env/logs",
			},
			expected: &DAGExecutionConfig{
				OnConflict:     "manual",
				BaseBranch:     "",
				MaxSpecRetries: 0,
				MaxLogSize:     "50MB",
				LogDir:         "/env/logs",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			clearDAGEnvVars()

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
			if result.LogDir != tt.expected.LogDir {
				t.Errorf("LogDir: got %q, want %q", result.LogDir, tt.expected.LogDir)
			}
		})
	}
}

// clearDAGEnvVars clears all DAG-related environment variables for testing.
func clearDAGEnvVars() {
	os.Unsetenv("AUTOSPEC_DAG_ON_CONFLICT")
	os.Unsetenv("AUTOSPEC_DAG_BASE_BRANCH")
	os.Unsetenv("AUTOSPEC_DAG_MAX_SPEC_RETRIES")
	os.Unsetenv("AUTOSPEC_DAG_MAX_LOG_SIZE")
	os.Unsetenv("AUTOSPEC_DAG_LOG_DIR")
	os.Unsetenv("AUTOSPEC_DAG_AUTOCOMMIT")
	os.Unsetenv("AUTOSPEC_DAG_AUTOCOMMIT_CMD")
	os.Unsetenv("AUTOSPEC_DAG_AUTOCOMMIT_RETRIES")
	os.Unsetenv("AUTOSPEC_DAG_AUTOMERGE")
}

func TestAutocommitConfig(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := map[string]struct {
		input           *DAGExecutionConfig
		envVars         map[string]string
		expectedEnabled bool
		expectedCmd     string
		expectedRetries int
	}{
		"defaults enabled": {
			input:           nil,
			envVars:         nil,
			expectedEnabled: true,
			expectedCmd:     "",
			expectedRetries: 1,
		},
		"explicit disabled via config": {
			input:           &DAGExecutionConfig{Autocommit: &falseVal},
			envVars:         nil,
			expectedEnabled: false,
			expectedCmd:     "",
			expectedRetries: 1,
		},
		"custom command": {
			input: &DAGExecutionConfig{
				AutocommitCmd:     "git add . && git commit -m 'auto'",
				AutocommitRetries: 3,
			},
			envVars:         nil,
			expectedEnabled: true,
			expectedCmd:     "git add . && git commit -m 'auto'",
			expectedRetries: 3,
		},
		"env var disables autocommit": {
			input: &DAGExecutionConfig{Autocommit: &trueVal},
			envVars: map[string]string{
				"AUTOSPEC_DAG_AUTOCOMMIT": "false",
			},
			expectedEnabled: false,
			expectedCmd:     "",
			expectedRetries: 1,
		},
		"env var enables autocommit": {
			input: &DAGExecutionConfig{Autocommit: &falseVal},
			envVars: map[string]string{
				"AUTOSPEC_DAG_AUTOCOMMIT": "true",
			},
			expectedEnabled: true,
			expectedCmd:     "",
			expectedRetries: 1,
		},
		"env var with 1 value": {
			input: nil,
			envVars: map[string]string{
				"AUTOSPEC_DAG_AUTOCOMMIT": "1",
			},
			expectedEnabled: true,
			expectedCmd:     "",
			expectedRetries: 1,
		},
		"env var overrides retries": {
			input: &DAGExecutionConfig{AutocommitRetries: 2},
			envVars: map[string]string{
				"AUTOSPEC_DAG_AUTOCOMMIT_RETRIES": "5",
			},
			expectedEnabled: true,
			expectedCmd:     "",
			expectedRetries: 5,
		},
		"env var retries clamped to max": {
			input: nil,
			envVars: map[string]string{
				"AUTOSPEC_DAG_AUTOCOMMIT_RETRIES": "20",
			},
			expectedEnabled: true,
			expectedCmd:     "",
			expectedRetries: 1, // Invalid value ignored, default used
		},
		"env var custom command": {
			input: nil,
			envVars: map[string]string{
				"AUTOSPEC_DAG_AUTOCOMMIT_CMD": "custom-commit-script",
			},
			expectedEnabled: true,
			expectedCmd:     "custom-commit-script",
			expectedRetries: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			clearDAGEnvVars()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := LoadDAGConfig(tt.input)

			if result.IsAutocommitEnabled() != tt.expectedEnabled {
				t.Errorf("IsAutocommitEnabled: got %v, want %v", result.IsAutocommitEnabled(), tt.expectedEnabled)
			}
			if result.AutocommitCmd != tt.expectedCmd {
				t.Errorf("AutocommitCmd: got %q, want %q", result.AutocommitCmd, tt.expectedCmd)
			}
			if result.GetAutocommitRetries() != tt.expectedRetries {
				t.Errorf("GetAutocommitRetries: got %d, want %d", result.GetAutocommitRetries(), tt.expectedRetries)
			}
		})
	}
}

func TestIsAutocommitEnabled(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := map[string]struct {
		cfg      *DAGExecutionConfig
		expected bool
	}{
		"nil Autocommit defaults to true": {
			cfg:      &DAGExecutionConfig{Autocommit: nil},
			expected: true,
		},
		"explicit true": {
			cfg:      &DAGExecutionConfig{Autocommit: &trueVal},
			expected: true,
		},
		"explicit false": {
			cfg:      &DAGExecutionConfig{Autocommit: &falseVal},
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.cfg.IsAutocommitEnabled(); got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetAutocommitRetries(t *testing.T) {
	tests := map[string]struct {
		retries  int
		expected int
	}{
		"zero": {
			retries:  0,
			expected: 0,
		},
		"valid value": {
			retries:  5,
			expected: 5,
		},
		"max value": {
			retries:  10,
			expected: 10,
		},
		"exceeds max clamped": {
			retries:  15,
			expected: 10,
		},
		"negative clamped to zero": {
			retries:  -1,
			expected: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &DAGExecutionConfig{AutocommitRetries: tt.retries}
			if got := cfg.GetAutocommitRetries(); got != tt.expected {
				t.Errorf("got %d, want %d", got, tt.expected)
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

func TestIsAutomergeEnabled(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := map[string]struct {
		cfg      *DAGExecutionConfig
		expected bool
	}{
		"nil Automerge defaults to true": {
			cfg:      &DAGExecutionConfig{Automerge: nil},
			expected: true,
		},
		"explicit true": {
			cfg:      &DAGExecutionConfig{Automerge: &trueVal},
			expected: true,
		},
		"explicit false": {
			cfg:      &DAGExecutionConfig{Automerge: &falseVal},
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.cfg.IsAutomergeEnabled(); got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := map[string]struct {
		cfg     *DAGExecutionConfig
		wantErr bool
		errMsg  string
	}{
		"both enabled - valid": {
			cfg:     &DAGExecutionConfig{Autocommit: &trueVal, Automerge: &trueVal},
			wantErr: false,
		},
		"both disabled - valid": {
			cfg:     &DAGExecutionConfig{Autocommit: &falseVal, Automerge: &falseVal},
			wantErr: false,
		},
		"autocommit enabled automerge disabled - valid": {
			cfg:     &DAGExecutionConfig{Autocommit: &trueVal, Automerge: &falseVal},
			wantErr: false,
		},
		"automerge enabled autocommit disabled - invalid": {
			cfg:     &DAGExecutionConfig{Autocommit: &falseVal, Automerge: &trueVal},
			wantErr: true,
			errMsg:  "automerge requires autocommit",
		},
		"automerge nil autocommit disabled - valid (automerge defaults true but ok)": {
			cfg:     &DAGExecutionConfig{Autocommit: &falseVal, Automerge: nil},
			wantErr: true,
			errMsg:  "automerge requires autocommit",
		},
		"both nil - valid (defaults)": {
			cfg:     &DAGExecutionConfig{Autocommit: nil, Automerge: nil},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.errMsg != "" && !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAutomergeEnvVar(t *testing.T) {
	tests := map[string]struct {
		envVal   string
		expected bool
	}{
		"true string": {
			envVal:   "true",
			expected: true,
		},
		"1 string": {
			envVal:   "1",
			expected: true,
		},
		"false string": {
			envVal:   "false",
			expected: false,
		},
		"0 string": {
			envVal:   "0",
			expected: false,
		},
		"random string defaults to false": {
			envVal:   "invalid",
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			clearDAGEnvVars()
			os.Setenv("AUTOSPEC_DAG_AUTOMERGE", tt.envVal)
			defer os.Unsetenv("AUTOSPEC_DAG_AUTOMERGE")

			result := LoadDAGConfig(nil)
			if result.IsAutomergeEnabled() != tt.expected {
				t.Errorf("IsAutomergeEnabled: got %v, want %v", result.IsAutomergeEnabled(), tt.expected)
			}
		})
	}
}

func TestAutomergeEnvOverridesConfig(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := map[string]struct {
		configVal *bool
		envVal    string
		expected  bool
	}{
		"env true overrides config false": {
			configVal: &falseVal,
			envVal:    "true",
			expected:  true,
		},
		"env false overrides config true": {
			configVal: &trueVal,
			envVal:    "false",
			expected:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			clearDAGEnvVars()
			os.Setenv("AUTOSPEC_DAG_AUTOMERGE", tt.envVal)
			defer os.Unsetenv("AUTOSPEC_DAG_AUTOMERGE")

			cfg := &DAGExecutionConfig{Automerge: tt.configVal}
			result := LoadDAGConfig(cfg)
			if result.IsAutomergeEnabled() != tt.expected {
				t.Errorf("IsAutomergeEnabled: got %v, want %v", result.IsAutomergeEnabled(), tt.expected)
			}
		})
	}
}
