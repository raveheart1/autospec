// Package completion_test tests shell configuration and completion block generation.
// Related: /home/ari/repos/autospec/internal/completion/shells.go
// Tags: completion, shell, bash, zsh, fish, powershell

package completion

import (
	"path/filepath"
	"testing"
)

func TestSupportedShells(t *testing.T) {
	shells := SupportedShells()
	if len(shells) != 4 {
		t.Errorf("expected 4 supported shells, got %d", len(shells))
	}

	expected := map[Shell]bool{
		Bash:       true,
		Zsh:        true,
		Fish:       true,
		PowerShell: true,
	}

	for _, shell := range shells {
		if !expected[shell] {
			t.Errorf("unexpected shell: %s", shell)
		}
	}
}

func TestIsValidShell(t *testing.T) {
	tests := map[string]struct {
		input string
		want  bool
	}{
		"bash lowercase":       {input: "bash", want: true},
		"bash uppercase":       {input: "BASH", want: true},
		"bash mixed case":      {input: "Bash", want: true},
		"zsh":                  {input: "zsh", want: true},
		"fish":                 {input: "fish", want: true},
		"powershell lowercase": {input: "powershell", want: true},
		"powershell mixed":     {input: "PowerShell", want: true},
		"invalid shell":        {input: "csh", want: false},
		"empty string":         {input: "", want: false},
		"random string":        {input: "notashell", want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := IsValidShell(tc.input)
			if got != tc.want {
				t.Errorf("IsValidShell(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestGetShellConfig(t *testing.T) {
	homeDir := "/home/testuser"

	tests := map[string]struct {
		shell             Shell
		wantRCPath        string
		wantCompletionDir string
		wantRequiresRCMod bool
	}{
		"bash config": {
			shell:             Bash,
			wantRCPath:        filepath.Join(homeDir, ".bashrc"),
			wantCompletionDir: "",
			wantRequiresRCMod: true,
		},
		"zsh config": {
			shell:             Zsh,
			wantRCPath:        filepath.Join(homeDir, ".zshrc"),
			wantCompletionDir: "",
			wantRequiresRCMod: true,
		},
		"fish config": {
			shell:             Fish,
			wantRCPath:        "",
			wantCompletionDir: filepath.Join(homeDir, ".config", "fish", "completions"),
			wantRequiresRCMod: false,
		},
		"powershell config": {
			shell:             PowerShell,
			wantRCPath:        filepath.Join(homeDir, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"),
			wantCompletionDir: "",
			wantRequiresRCMod: true,
		},
		"unknown shell": {
			shell:             Shell("unknown"),
			wantRCPath:        "",
			wantCompletionDir: "",
			wantRequiresRCMod: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			config := GetShellConfig(tc.shell, homeDir)

			if config.Shell != tc.shell && tc.shell != Shell("unknown") {
				t.Errorf("Shell = %v, want %v", config.Shell, tc.shell)
			}
			if config.RCPath != tc.wantRCPath {
				t.Errorf("RCPath = %v, want %v", config.RCPath, tc.wantRCPath)
			}
			if config.CompletionDir != tc.wantCompletionDir {
				t.Errorf("CompletionDir = %v, want %v", config.CompletionDir, tc.wantCompletionDir)
			}
			if config.RequiresRCModification != tc.wantRequiresRCMod {
				t.Errorf("RequiresRCModification = %v, want %v", config.RequiresRCModification, tc.wantRequiresRCMod)
			}
		})
	}
}

func TestDetectShell(t *testing.T) {
	tests := map[string]struct {
		shellEnv  string
		wantShell Shell
		wantErr   bool
	}{
		"bash full path":     {shellEnv: "/bin/bash", wantShell: Bash, wantErr: false},
		"bash usr path":      {shellEnv: "/usr/bin/bash", wantShell: Bash, wantErr: false},
		"zsh full path":      {shellEnv: "/bin/zsh", wantShell: Zsh, wantErr: false},
		"zsh usr local path": {shellEnv: "/usr/local/bin/zsh", wantShell: Zsh, wantErr: false},
		"fish path":          {shellEnv: "/usr/bin/fish", wantShell: Fish, wantErr: false},
		"pwsh path":          {shellEnv: "/usr/bin/pwsh", wantShell: PowerShell, wantErr: false},
		"powershell path":    {shellEnv: "/usr/bin/powershell", wantShell: PowerShell, wantErr: false},
		"unsupported shell":  {shellEnv: "/bin/csh", wantShell: "", wantErr: true},
		"empty shell env":    {shellEnv: "", wantShell: "", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Set the environment variable for this test
			t.Setenv("SHELL", tc.shellEnv)

			shell, err := DetectShell()

			if (err != nil) != tc.wantErr {
				t.Errorf("DetectShell() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if shell != tc.wantShell {
				t.Errorf("DetectShell() = %v, want %v", shell, tc.wantShell)
			}
		})
	}
}

func TestGetCompletionBlock(t *testing.T) {
	tests := map[string]struct {
		shell             Shell
		wantEmpty         bool
		wantStartMarker   string
		wantEndMarker     string
		wantContentSubstr string
	}{
		"bash block": {
			shell:             Bash,
			wantEmpty:         false,
			wantStartMarker:   StartMarker,
			wantEndMarker:     EndMarker,
			wantContentSubstr: "source <(autospec completion bash)",
		},
		"zsh block": {
			shell:             Zsh,
			wantEmpty:         false,
			wantStartMarker:   StartMarker,
			wantEndMarker:     EndMarker,
			wantContentSubstr: "autoload -U compinit && compinit",
		},
		"powershell block": {
			shell:             PowerShell,
			wantEmpty:         false,
			wantStartMarker:   StartMarker,
			wantEndMarker:     EndMarker,
			wantContentSubstr: "autospec completion powershell | Out-String | Invoke-Expression",
		},
		"fish block (empty)": {
			shell:     Fish,
			wantEmpty: true,
		},
		"unknown shell (empty)": {
			shell:     Shell("unknown"),
			wantEmpty: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			block := GetCompletionBlock(tc.shell)

			if block.IsEmpty() != tc.wantEmpty {
				t.Errorf("IsEmpty() = %v, want %v", block.IsEmpty(), tc.wantEmpty)
			}

			if !tc.wantEmpty {
				if block.StartMarker != tc.wantStartMarker {
					t.Errorf("StartMarker = %v, want %v", block.StartMarker, tc.wantStartMarker)
				}
				if block.EndMarker != tc.wantEndMarker {
					t.Errorf("EndMarker = %v, want %v", block.EndMarker, tc.wantEndMarker)
				}
				if tc.wantContentSubstr != "" && block.Content != "" {
					if !containsString(block.Content, tc.wantContentSubstr) {
						t.Errorf("Content = %q, want to contain %q", block.Content, tc.wantContentSubstr)
					}
				}
			}
		})
	}
}

func TestCompletionBlockFormatBlock(t *testing.T) {
	tests := map[string]struct {
		block      CompletionBlock
		wantSubstr []string
		wantEmpty  bool
	}{
		"bash formatted block": {
			block: GetCompletionBlock(Bash),
			wantSubstr: []string{
				StartMarker,
				EndMarker,
				"source <(autospec completion bash)",
			},
			wantEmpty: false,
		},
		"empty block": {
			block:     CompletionBlock{},
			wantEmpty: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			formatted := tc.block.FormatBlock()

			if tc.wantEmpty {
				if formatted != "" {
					t.Errorf("FormatBlock() = %q, want empty string", formatted)
				}
				return
			}

			for _, substr := range tc.wantSubstr {
				if !containsString(formatted, substr) {
					t.Errorf("FormatBlock() = %q, want to contain %q", formatted, substr)
				}
			}
		})
	}
}

func TestMarkerConstants(t *testing.T) {
	// Verify markers match FR-007 specification
	expectedStart := "# >>> autospec completion >>>"
	expectedEnd := "# <<< autospec completion <<<"

	if StartMarker != expectedStart {
		t.Errorf("StartMarker = %q, want %q", StartMarker, expectedStart)
	}
	if EndMarker != expectedEnd {
		t.Errorf("EndMarker = %q, want %q", EndMarker, expectedEnd)
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
