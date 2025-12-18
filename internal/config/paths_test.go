// Package config_test tests configuration path resolution and XDG compliance.
// Related: internal/config/paths.go
// Tags: config, paths, xdg, user-config, project-config, legacy
package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUserConfigPath(t *testing.T) {
	path, err := UserConfigPath()
	if err != nil {
		t.Fatalf("UserConfigPath() returned error: %v", err)
	}

	// Should end with autospec/config.yml
	if !strings.HasSuffix(path, filepath.Join("autospec", "config.yml")) {
		t.Errorf("UserConfigPath() = %q, want path ending with autospec/config.yml", path)
	}

	// Should be an absolute path
	if !filepath.IsAbs(path) {
		t.Errorf("UserConfigPath() = %q, want absolute path", path)
	}
}

func TestUserConfigPath_XDGConfigHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("XDG_CONFIG_HOME is only used on Linux")
	}

	// Save original value
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	// Set custom XDG_CONFIG_HOME
	customDir := "/custom/config"
	os.Setenv("XDG_CONFIG_HOME", customDir)

	path, err := UserConfigPath()
	if err != nil {
		t.Fatalf("UserConfigPath() returned error: %v", err)
	}

	expected := filepath.Join(customDir, "autospec", "config.yml")
	if path != expected {
		t.Errorf("UserConfigPath() = %q, want %q", path, expected)
	}
}

func TestUserConfigDir(t *testing.T) {
	dir, err := UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() returned error: %v", err)
	}

	// Should end with autospec
	if !strings.HasSuffix(dir, "autospec") {
		t.Errorf("UserConfigDir() = %q, want path ending with autospec", dir)
	}

	// Should be an absolute path
	if !filepath.IsAbs(dir) {
		t.Errorf("UserConfigDir() = %q, want absolute path", dir)
	}
}

func TestProjectConfigPath(t *testing.T) {
	path := ProjectConfigPath()
	expected := filepath.Join(".autospec", "config.yml")
	if path != expected {
		t.Errorf("ProjectConfigPath() = %q, want %q", path, expected)
	}
}

func TestProjectConfigDir(t *testing.T) {
	dir := ProjectConfigDir()
	if dir != ".autospec" {
		t.Errorf("ProjectConfigDir() = %q, want %q", dir, ".autospec")
	}
}

func TestLegacyUserConfigPath(t *testing.T) {
	path, err := LegacyUserConfigPath()
	if err != nil {
		t.Fatalf("LegacyUserConfigPath() returned error: %v", err)
	}

	// Should end with .autospec/config.json
	if !strings.HasSuffix(path, filepath.Join(".autospec", "config.json")) {
		t.Errorf("LegacyUserConfigPath() = %q, want path ending with .autospec/config.json", path)
	}

	// Should be an absolute path
	if !filepath.IsAbs(path) {
		t.Errorf("LegacyUserConfigPath() = %q, want absolute path", path)
	}
}

func TestLegacyProjectConfigPath(t *testing.T) {
	path := LegacyProjectConfigPath()
	expected := filepath.Join(".autospec", "config.json")
	if path != expected {
		t.Errorf("LegacyProjectConfigPath() = %q, want %q", path, expected)
	}
}

func TestLegacyGlobalConfigPath(t *testing.T) {
	path, err := LegacyGlobalConfigPath()
	if err != nil {
		t.Fatalf("LegacyGlobalConfigPath() returned error: %v", err)
	}

	// Should be same as LegacyUserConfigPath
	legacyUser, _ := LegacyUserConfigPath()
	if path != legacyUser {
		t.Errorf("LegacyGlobalConfigPath() = %q, want %q (same as LegacyUserConfigPath)", path, legacyUser)
	}
}
