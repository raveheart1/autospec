// Package completion provides functionality for installing shell completions for autospec.
package completion

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Shell represents a supported shell type
type Shell string

const (
	// Bash shell
	Bash Shell = "bash"
	// Zsh shell
	Zsh Shell = "zsh"
	// Fish shell
	Fish Shell = "fish"
	// PowerShell
	PowerShell Shell = "powershell"
)

// SupportedShells returns the list of all supported shell types
func SupportedShells() []Shell {
	return []Shell{Bash, Zsh, Fish, PowerShell}
}

// IsValidShell checks if the given string is a valid shell type
func IsValidShell(s string) bool {
	switch Shell(strings.ToLower(s)) {
	case Bash, Zsh, Fish, PowerShell:
		return true
	default:
		return false
	}
}

// ShellConfig contains configuration for a specific shell's completion setup
type ShellConfig struct {
	// Shell is the shell type
	Shell Shell
	// RCPath is the path to the rc file (empty for fish)
	RCPath string
	// CompletionDir is the path to the completion directory (fish only)
	CompletionDir string
	// RequiresRCModification indicates whether the shell needs rc file modification
	RequiresRCModification bool
}

// GetShellConfig returns the configuration for the specified shell type.
// The homeDir parameter is used as the base for user-specific paths.
func GetShellConfig(shell Shell, homeDir string) ShellConfig {
	switch shell {
	case Bash:
		return ShellConfig{
			Shell:                  Bash,
			RCPath:                 filepath.Join(homeDir, ".bashrc"),
			CompletionDir:          "",
			RequiresRCModification: true,
		}
	case Zsh:
		return ShellConfig{
			Shell:                  Zsh,
			RCPath:                 filepath.Join(homeDir, ".zshrc"),
			CompletionDir:          "",
			RequiresRCModification: true,
		}
	case Fish:
		return ShellConfig{
			Shell:                  Fish,
			RCPath:                 "",
			CompletionDir:          filepath.Join(homeDir, ".config", "fish", "completions"),
			RequiresRCModification: false,
		}
	case PowerShell:
		// PowerShell profile location varies by OS
		profilePath := getPowerShellProfilePath(homeDir)
		return ShellConfig{
			Shell:                  PowerShell,
			RCPath:                 profilePath,
			CompletionDir:          "",
			RequiresRCModification: true,
		}
	default:
		return ShellConfig{}
	}
}

// getPowerShellProfilePath returns the PowerShell profile path for the current OS
func getPowerShellProfilePath(homeDir string) string {
	if runtime.GOOS == "windows" {
		// Windows: Documents/WindowsPowerShell/Microsoft.PowerShell_profile.ps1
		return filepath.Join(homeDir, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	}
	// Linux/macOS: ~/.config/powershell/Microsoft.PowerShell_profile.ps1
	return filepath.Join(homeDir, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
}

// DetectShell detects the current shell from the $SHELL environment variable.
// Returns an error if $SHELL is not set or the shell is not recognized.
func DetectShell() (Shell, error) {
	shellEnv := os.Getenv("SHELL")
	if shellEnv == "" {
		// On Windows, check for PowerShell
		if runtime.GOOS == "windows" {
			return PowerShell, nil
		}
		return "", fmt.Errorf("$SHELL environment variable is not set; please specify a shell: bash, zsh, fish, or powershell")
	}

	// Extract the shell name from the path (e.g., /bin/bash -> bash)
	shellName := strings.ToLower(filepath.Base(shellEnv))

	switch shellName {
	case "bash":
		return Bash, nil
	case "zsh":
		return Zsh, nil
	case "fish":
		return Fish, nil
	case "pwsh", "powershell":
		return PowerShell, nil
	default:
		return "", fmt.Errorf("unsupported shell: %s; supported shells are: bash, zsh, fish, powershell", shellName)
	}
}

// Completion block markers per FR-007
const (
	// StartMarker marks the beginning of the autospec completion block
	StartMarker = "# >>> autospec completion >>>"
	// EndMarker marks the end of the autospec completion block
	EndMarker = "# <<< autospec completion <<<"
)

// CompletionBlock represents a shell-specific completion sourcing block
type CompletionBlock struct {
	// StartMarker is the block start marker
	StartMarker string
	// EndMarker is the block end marker
	EndMarker string
	// Content is the shell-specific sourcing commands
	Content string
}

// GetCompletionBlock returns the completion block for the specified shell type.
// For fish, returns an empty block since fish uses standalone completion files.
func GetCompletionBlock(shell Shell) CompletionBlock {
	switch shell {
	case Bash:
		return CompletionBlock{
			StartMarker: StartMarker,
			EndMarker:   EndMarker,
			Content:     "source <(autospec completion bash)",
		}
	case Zsh:
		return CompletionBlock{
			StartMarker: StartMarker,
			EndMarker:   EndMarker,
			Content:     "autoload -U compinit && compinit\nsource <(autospec completion zsh)",
		}
	case PowerShell:
		return CompletionBlock{
			StartMarker: StartMarker,
			EndMarker:   EndMarker,
			Content:     "autospec completion powershell | Out-String | Invoke-Expression",
		}
	case Fish:
		// Fish uses standalone completion files, no rc block needed
		return CompletionBlock{}
	default:
		return CompletionBlock{}
	}
}

// FormatBlock formats the completion block with markers for writing to an rc file
func (b CompletionBlock) FormatBlock() string {
	if b.Content == "" {
		return ""
	}
	return fmt.Sprintf("\n%s\n%s\n%s\n", b.StartMarker, b.Content, b.EndMarker)
}

// IsEmpty returns true if the completion block has no content
func (b CompletionBlock) IsEmpty() bool {
	return b.Content == ""
}
