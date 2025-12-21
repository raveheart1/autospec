package update

import (
	"fmt"
	"os"
	"path/filepath"
)

// Installer handles binary installation with backup and rollback.
type Installer struct {
	executablePath string
	backupPath     string
}

// NewInstaller creates a new installer for the current executable.
func NewInstaller() (*Installer, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("determining executable path: %w", err)
	}

	// Resolve any symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		// If symlink resolution fails, use original path
		realPath = execPath
	}

	return &Installer{
		executablePath: realPath,
		backupPath:     realPath + ".bak",
	}, nil
}

// GetExecutablePath returns the path to the current executable.
func (i *Installer) GetExecutablePath() string {
	return i.executablePath
}

// GetBackupPath returns the path where the backup will be stored.
func (i *Installer) GetBackupPath() string {
	return i.backupPath
}

// CreateBackup renames the current binary to .bak extension.
func (i *Installer) CreateBackup() error {
	// Remove any existing backup
	if err := os.Remove(i.backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing old backup: %w", err)
	}

	if err := os.Rename(i.executablePath, i.backupPath); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	return nil
}

// InstallBinary moves a new binary into place.
func (i *Installer) InstallBinary(newBinaryPath string) error {
	if err := os.Rename(newBinaryPath, i.executablePath); err != nil {
		return fmt.Errorf("installing new binary: %w", err)
	}
	return nil
}

// SetPermissions ensures the binary is executable.
func (i *Installer) SetPermissions() error {
	if err := os.Chmod(i.executablePath, 0755); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}
	return nil
}

// Rollback restores the backup if something goes wrong.
func (i *Installer) Rollback() error {
	// Check if backup exists
	if _, err := os.Stat(i.backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup found to restore")
	}

	// Remove the failed new binary if it exists
	if err := os.Remove(i.executablePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing failed binary: %w", err)
	}

	// Restore backup
	if err := os.Rename(i.backupPath, i.executablePath); err != nil {
		return fmt.Errorf("restoring backup: %w", err)
	}

	return nil
}

// CleanupBackup removes the backup file after successful installation.
func (i *Installer) CleanupBackup() error {
	if err := os.Remove(i.backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cleaning up backup: %w", err)
	}
	return nil
}

// CheckWritePermission checks if we have write access to the executable location.
func (i *Installer) CheckWritePermission() error {
	dir := filepath.Dir(i.executablePath)

	// Try to create a temporary file in the directory
	tmpFile, err := os.CreateTemp(dir, ".autospec-write-test-*")
	if err != nil {
		return fmt.Errorf("no write permission to %s: %w", dir, err)
	}

	tmpFile.Close()
	os.Remove(tmpFile.Name())

	return nil
}
