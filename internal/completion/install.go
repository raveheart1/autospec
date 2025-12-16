package completion

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// InstallAction describes the action taken during installation
type InstallAction string

const (
	// ActionInstalled means a new completion block was added
	ActionInstalled InstallAction = "installed"
	// ActionUpdated means an existing completion block was updated
	ActionUpdated InstallAction = "updated"
	// ActionSkipped means the installation was skipped (already exists)
	ActionSkipped InstallAction = "skipped"
)

// InstallResult represents the outcome of an installation operation
type InstallResult struct {
	// Success indicates whether installation succeeded
	Success bool
	// BackupPath is the path to the backup file if created (empty for fish or if no backup needed)
	BackupPath string
	// ConfigPath is the path to the modified config file
	ConfigPath string
	// Action is what action was taken
	Action InstallAction
	// Message is a human-readable status message
	Message string
	// Shell is the shell type that was installed
	Shell Shell
}

// PermissionError indicates a permission-related failure during installation
type PermissionError struct {
	Path      string
	Operation string
	Err       error
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("permission denied: cannot %s %s: %v", e.Operation, e.Path, e.Err)
}

func (e *PermissionError) Unwrap() error {
	return e.Err
}

// IsPermissionError checks if an error is a PermissionError
func IsPermissionError(err error) bool {
	var pe *PermissionError
	return errors.As(err, &pe)
}

// CreateBackup creates a timestamped backup of the specified file.
// Returns the backup path on success, or empty string if the original file doesn't exist.
// The backup format is: original.autospec-backup-YYYYMMDD-HHMMSS
func CreateBackup(filePath string) (string, error) {
	// Check if original file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Original doesn't exist, no backup needed
		return "", nil
	}

	// Read original content
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsPermission(err) {
			return "", &PermissionError{Path: filePath, Operation: "read", Err: err}
		}
		return "", fmt.Errorf("failed to read file for backup: %w", err)
	}

	// Generate timestamped backup filename
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.autospec-backup-%s", filePath, timestamp)

	// Write backup file
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		if os.IsPermission(err) {
			return "", &PermissionError{Path: backupPath, Operation: "write", Err: err}
		}
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

// HasExistingInstallation checks if the completion block is already installed in the rc file.
// Returns true if the start marker is found in the file content.
func HasExistingInstallation(filePath string) (bool, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsPermission(err) {
			return false, &PermissionError{Path: filePath, Operation: "read", Err: err}
		}
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), StartMarker) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	return false, nil
}

// Install orchestrates the full installation flow for the specified shell.
// It handles all four shell types with appropriate behavior:
// - Bash/Zsh/PowerShell: Creates backup, appends completion block to rc file
// - Fish: Writes completion script directly to completions directory
func Install(shell Shell) (*InstallResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	config := GetShellConfig(shell, homeDir)

	// Handle fish specially - writes to completions directory
	if shell == Fish {
		return installFish(config)
	}

	// For bash/zsh/powershell - modify rc file
	return installRCFile(shell, config)
}

// installFish handles fish shell completion installation.
// Fish uses a standalone completion file in ~/.config/fish/completions/
func installFish(config ShellConfig) (*InstallResult, error) {
	completionPath := filepath.Join(config.CompletionDir, "autospec.fish")

	// Create completions directory if it doesn't exist
	if err := os.MkdirAll(config.CompletionDir, 0755); err != nil {
		if os.IsPermission(err) {
			return nil, &PermissionError{Path: config.CompletionDir, Operation: "create directory", Err: err}
		}
		return nil, fmt.Errorf("failed to create fish completions directory: %w", err)
	}

	// Generate fish completions using autospec completion fish command
	completionScript, err := generateCompletionScript(Fish)
	if err != nil {
		return nil, fmt.Errorf("failed to generate fish completion script: %w", err)
	}

	// Check if file already exists
	action := ActionInstalled
	if _, err := os.Stat(completionPath); err == nil {
		action = ActionUpdated
	}

	// Write completion file
	if err := os.WriteFile(completionPath, []byte(completionScript), 0644); err != nil {
		if os.IsPermission(err) {
			return nil, &PermissionError{Path: completionPath, Operation: "write", Err: err}
		}
		return nil, fmt.Errorf("failed to write fish completion file: %w", err)
	}

	msg := fmt.Sprintf("Fish completions %s at %s", action, completionPath)
	if action == ActionInstalled {
		msg += "\nCompletions will be available in new shell sessions."
	}

	return &InstallResult{
		Success:    true,
		BackupPath: "", // Fish doesn't need backups
		ConfigPath: completionPath,
		Action:     action,
		Message:    msg,
		Shell:      Fish,
	}, nil
}

// installRCFile handles rc file modification for bash/zsh/powershell
func installRCFile(shell Shell, config ShellConfig) (*InstallResult, error) {
	rcPath := config.RCPath

	// Check for existing installation
	exists, err := HasExistingInstallation(rcPath)
	if err != nil {
		return nil, err
	}

	if exists {
		// Already installed, could offer to update/replace
		return &InstallResult{
			Success:    true,
			BackupPath: "",
			ConfigPath: rcPath,
			Action:     ActionSkipped,
			Message:    fmt.Sprintf("Completion already installed in %s\nTo reinstall, remove the existing block between the markers first.", rcPath),
			Shell:      shell,
		}, nil
	}

	// Create backup before modification
	backupPath, err := CreateBackup(rcPath)
	if err != nil {
		return nil, err
	}

	// Get completion block for this shell
	block := GetCompletionBlock(shell)
	formattedBlock := block.FormatBlock()

	// Ensure parent directory exists (for PowerShell profile which may not exist)
	if err := os.MkdirAll(filepath.Dir(rcPath), 0755); err != nil {
		if os.IsPermission(err) {
			return nil, &PermissionError{Path: filepath.Dir(rcPath), Operation: "create directory", Err: err}
		}
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Append completion block to rc file
	file, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsPermission(err) {
			return nil, &PermissionError{Path: rcPath, Operation: "write", Err: err}
		}
		return nil, fmt.Errorf("failed to open rc file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(formattedBlock); err != nil {
		if os.IsPermission(err) {
			return nil, &PermissionError{Path: rcPath, Operation: "write", Err: err}
		}
		return nil, fmt.Errorf("failed to write completion block: %w", err)
	}

	action := ActionInstalled
	msg := fmt.Sprintf("Completions installed in %s", rcPath)
	if backupPath != "" {
		msg += fmt.Sprintf("\nBackup created at %s", backupPath)
	}
	msg += "\n\nTo activate completions, run:"
	msg += fmt.Sprintf("\n  source %s", rcPath)
	msg += "\nOr start a new shell session."

	return &InstallResult{
		Success:    true,
		BackupPath: backupPath,
		ConfigPath: rcPath,
		Action:     action,
		Message:    msg,
		Shell:      shell,
	}, nil
}

// generateCompletionScript generates the completion script for a given shell
// by running autospec completion <shell>
func generateCompletionScript(shell Shell) (string, error) {
	// Get the path to the current executable
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Run autospec completion <shell>
	cmd := exec.Command(exePath, "completion", string(shell))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to generate completion script: %w", err)
	}

	return string(output), nil
}

// GetManualInstructions returns manual installation instructions for the specified shell.
func GetManualInstructions(shell Shell) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Manual installation instructions for %s:\n\n", shell))

	switch shell {
	case Bash:
		sb.WriteString("Add the following to your ~/.bashrc:\n\n")
		sb.WriteString("  " + StartMarker + "\n")
		sb.WriteString("  source <(autospec completion bash)\n")
		sb.WriteString("  " + EndMarker + "\n")
		sb.WriteString("\nThen run: source ~/.bashrc\n")

	case Zsh:
		sb.WriteString("Add the following to your ~/.zshrc:\n\n")
		sb.WriteString("  " + StartMarker + "\n")
		sb.WriteString("  autoload -U compinit && compinit\n")
		sb.WriteString("  source <(autospec completion zsh)\n")
		sb.WriteString("  " + EndMarker + "\n")
		sb.WriteString("\nThen run: source ~/.zshrc\n")

	case Fish:
		sb.WriteString("Run the following command:\n\n")
		sb.WriteString("  autospec completion fish > ~/.config/fish/completions/autospec.fish\n")
		sb.WriteString("\nCompletions will be available in new shell sessions.\n")

	case PowerShell:
		sb.WriteString("Add the following to your PowerShell profile ($PROFILE):\n\n")
		sb.WriteString("  " + StartMarker + "\n")
		sb.WriteString("  autospec completion powershell | Out-String | Invoke-Expression\n")
		sb.WriteString("  " + EndMarker + "\n")
		sb.WriteString("\nTo find your profile location, run: echo $PROFILE\n")
		sb.WriteString("Then reload your profile or start a new PowerShell session.\n")
	}

	return sb.String()
}

// GetAllManualInstructions returns manual installation instructions for all supported shells.
func GetAllManualInstructions(out io.Writer) {
	for _, shell := range SupportedShells() {
		fmt.Fprintln(out, GetManualInstructions(shell))
		fmt.Fprintln(out, strings.Repeat("-", 50))
		fmt.Fprintln(out)
	}
}
