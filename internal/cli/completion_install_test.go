package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionInstallCommand_ManualFlag(t *testing.T) {
	tests := map[string]struct {
		args           []string
		wantContains   []string
		wantNotContain []string
	}{
		"manual flag with auto-detect": {
			args: []string{"completion", "install", "--manual"},
			wantContains: []string{
				"Manual installation instructions",
				"# >>> autospec completion >>>",
			},
		},
		"manual flag with bash": {
			args: []string{"completion", "install", "bash", "--manual"},
			wantContains: []string{
				"Manual installation instructions for bash",
				".bashrc",
				"source <(autospec completion bash)",
				"# >>> autospec completion >>>",
			},
		},
		"manual flag with zsh": {
			args: []string{"completion", "install", "zsh", "--manual"},
			wantContains: []string{
				"Manual installation instructions for zsh",
				".zshrc",
				"compinit",
			},
		},
		"manual flag with fish": {
			args: []string{"completion", "install", "fish", "--manual"},
			wantContains: []string{
				"Manual installation instructions for fish",
				"completions/autospec.fish",
			},
		},
		"manual flag with powershell": {
			args: []string{"completion", "install", "powershell", "--manual"},
			wantContains: []string{
				"Manual installation instructions for powershell",
				"$PROFILE",
				"Out-String",
				"Invoke-Expression",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Set a known shell for consistent detection
			t.Setenv("SHELL", "/bin/zsh")

			cmd := rootCmd
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			output := buf.String()
			for _, want := range tc.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output = %q, want to contain %q", output, want)
				}
			}
			for _, notWant := range tc.wantNotContain {
				if strings.Contains(output, notWant) {
					t.Errorf("output = %q, should not contain %q", output, notWant)
				}
			}
		})
	}
}

func TestCompletionInstallCommand_InvalidShell(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"completion", "install", "invalidshell"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() should have returned an error for invalid shell")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "unknown shell") {
		t.Errorf("error = %q, want to contain 'unknown shell'", errStr)
	}
	if !strings.Contains(errStr, "bash, zsh, fish, powershell") {
		t.Errorf("error = %q, want to contain supported shells list", errStr)
	}
}

func TestCompletionInstallCommand_ShellDetection(t *testing.T) {
	tests := map[string]struct {
		shellEnv     string
		wantContains []string
	}{
		"bash detection": {
			shellEnv:     "/bin/bash",
			wantContains: []string{"Detected shell: bash"},
		},
		"zsh detection": {
			shellEnv:     "/usr/bin/zsh",
			wantContains: []string{"Detected shell: zsh"},
		},
		"fish detection": {
			shellEnv:     "/usr/bin/fish",
			wantContains: []string{"Detected shell: fish"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("SHELL", tc.shellEnv)

			cmd := rootCmd
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs([]string{"completion", "install", "--manual"})

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			output := buf.String()
			for _, want := range tc.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output = %q, want to contain %q", output, want)
				}
			}
		})
	}
}

func TestCompletionInstallCommand_EmptyShellEnv(t *testing.T) {
	t.Setenv("SHELL", "")

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"completion", "install", "--manual"})

	err := cmd.Execute()
	// Should not error, just provide guidance
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Could not auto-detect shell") {
		t.Errorf("output = %q, want to contain auto-detect failure message", output)
	}
	if !strings.Contains(output, "autospec completion install bash") {
		t.Errorf("output = %q, want to contain shell specification examples", output)
	}
}

func TestCompletionInstallHelp(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"completion", "install", "--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"Install shell completions for autospec",
		"auto-detects your shell",
		"--manual",
		"bash",
		"zsh",
		"fish",
		"powershell",
	}

	for _, want := range expectedStrings {
		if !strings.Contains(output, want) {
			t.Errorf("help output = %q, want to contain %q", output, want)
		}
	}
}

func TestCompletionShellGeneration(t *testing.T) {
	// Test that the shell-specific commands generate valid output
	shells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			cmd := rootCmd
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs([]string{"completion", shell})

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			output := buf.String()
			if len(output) < 100 {
				t.Errorf("completion output too short for %s: len=%d", shell, len(output))
			}
			// Each completion script should mention autospec
			if !strings.Contains(output, "autospec") {
				t.Errorf("completion output for %s should contain 'autospec'", shell)
			}
		})
	}
}
