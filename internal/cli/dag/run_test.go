package dag

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCmd_ValidateFileArg(t *testing.T) {
	tmpDir := t.TempDir()

	validFile := filepath.Join(tmpDir, "valid.yaml")
	if err := os.WriteFile(validFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	tests := map[string]struct {
		path        string
		expectError bool
		errContains string
	}{
		"valid file": {
			path:        validFile,
			expectError: false,
		},
		"nonexistent file": {
			path:        filepath.Join(tmpDir, "nonexistent.yaml"),
			expectError: true,
			errContains: "file not found",
		},
		"directory instead of file": {
			path:        tmpDir,
			expectError: true,
			errContains: "directory",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateFileArg(tt.path)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectError && tt.errContains != "" && err != nil {
				if !bytes.Contains([]byte(err.Error()), []byte(tt.errContains)) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestRunCmd_DryRunFlagParsing(t *testing.T) {
	tests := map[string]struct {
		args         []string
		expectDryRun bool
		expectForce  bool
	}{
		"no flags": {
			args:         []string{"file.yaml"},
			expectDryRun: false,
			expectForce:  false,
		},
		"dry-run enabled": {
			args:         []string{"file.yaml", "--dry-run"},
			expectDryRun: true,
			expectForce:  false,
		},
		"force enabled": {
			args:         []string{"file.yaml", "--force"},
			expectDryRun: false,
			expectForce:  true,
		},
		"both flags": {
			args:         []string{"file.yaml", "--dry-run", "--force"},
			expectDryRun: true,
			expectForce:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset flags for each test
			runCmd.Flags().Set("dry-run", "false")
			runCmd.Flags().Set("force", "false")

			if err := runCmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			dryRun, _ := runCmd.Flags().GetBool("dry-run")
			force, _ := runCmd.Flags().GetBool("force")

			if dryRun != tt.expectDryRun {
				t.Errorf("expected dry-run=%v, got %v", tt.expectDryRun, dryRun)
			}
			if force != tt.expectForce {
				t.Errorf("expected force=%v, got %v", tt.expectForce, force)
			}
		})
	}
}

func TestFormatDagValidationErrors(t *testing.T) {
	tests := map[string]struct {
		errs      []error
		wantCount int
	}{
		"single error": {
			errs:      []error{os.ErrNotExist},
			wantCount: 1,
		},
		"multiple errors": {
			errs:      []error{os.ErrNotExist, os.ErrPermission},
			wantCount: 2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := formatDagValidationErrors("test.yaml", tt.errs)
			if err == nil {
				t.Error("expected error but got nil")
			}
		})
	}
}

func TestPrintRunSuccess(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRunSuccess("test-run-id")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("test-run-id")) {
		t.Error("expected output to contain run ID")
	}
}

func TestPrintRunFailure(t *testing.T) {
	tests := map[string]struct {
		runID string
	}{
		"with run ID": {
			runID: "test-run-id",
		},
		"without run ID": {
			runID: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := printRunFailure(tt.runID, os.ErrNotExist)
			if err == nil {
				t.Error("expected error to be returned")
			}
		})
	}
}

func TestRunCmd_CommandDefinition(t *testing.T) {
	tests := map[string]struct {
		checkFunc func() bool
		desc      string
	}{
		"has use": {
			checkFunc: func() bool { return runCmd.Use != "" },
			desc:      "command should have Use field set",
		},
		"has short description": {
			checkFunc: func() bool { return runCmd.Short != "" },
			desc:      "command should have Short description",
		},
		"has long description": {
			checkFunc: func() bool { return runCmd.Long != "" },
			desc:      "command should have Long description",
		},
		"has example": {
			checkFunc: func() bool { return runCmd.Example != "" },
			desc:      "command should have Example",
		},
		"requires exactly one arg": {
			checkFunc: func() bool { return runCmd.Args != nil },
			desc:      "command should have Args validator",
		},
		"has RunE function": {
			checkFunc: func() bool { return runCmd.RunE != nil },
			desc:      "command should have RunE function",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if !tt.checkFunc() {
				t.Error(tt.desc)
			}
		})
	}
}

func TestRunCmd_Flags(t *testing.T) {
	tests := map[string]struct {
		flagName string
		flagType string
	}{
		"dry-run flag": {
			flagName: "dry-run",
			flagType: "bool",
		},
		"force flag": {
			flagName: "force",
			flagType: "bool",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			flag := runCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("expected flag %q to exist", tt.flagName)
				return
			}
			if flag.Value.Type() != tt.flagType {
				t.Errorf("expected flag type %q, got %q", tt.flagType, flag.Value.Type())
			}
		})
	}
}
