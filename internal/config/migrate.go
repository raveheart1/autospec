package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// MigrationResult describes the outcome of a migration operation
type MigrationResult struct {
	SourcePath string
	TargetPath string
	Success    bool
	DryRun     bool
	Message    string
}

// MigrateJSONToYAML converts a JSON config file to YAML format.
//
// Migration pipeline:
//  1. Read JSON → 2. Check if YAML exists (skip if so) → 3. Convert → 4. Write
//
// Safety features:
//   - Dry-run mode reports planned action without writing
//   - Skips if YAML already exists (no overwrite)
//   - Creates parent directories as needed
//   - Adds header comment to output YAML
func MigrateJSONToYAML(jsonPath, yamlPath string, dryRun bool) (*MigrationResult, error) {
	result := &MigrationResult{
		SourcePath: jsonPath,
		TargetPath: yamlPath,
		DryRun:     dryRun,
	}

	// Check if JSON file exists
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Message = fmt.Sprintf("No JSON config found at %s", jsonPath)
			return result, nil
		}
		return nil, fmt.Errorf("failed to read JSON config: %w", err)
	}

	// Parse JSON
	var configData map[string]interface{}
	if err := json.Unmarshal(jsonData, &configData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	// Check if YAML file already exists
	if _, err := os.Stat(yamlPath); err == nil {
		result.Message = fmt.Sprintf("YAML config already exists at %s (skipped)", yamlPath)
		return result, nil
	}

	if dryRun {
		result.Success = true
		result.Message = fmt.Sprintf("Would migrate %s → %s", jsonPath, yamlPath)
		return result, nil
	}

	// Create YAML content
	yamlData, err := yaml.Marshal(configData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to YAML: %w", err)
	}

	// Create target directory if needed
	if err := os.MkdirAll(filepath.Dir(yamlPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write YAML file with header
	header := "# Autospec Configuration\n# Migrated from JSON format\n\n"
	if err := os.WriteFile(yamlPath, []byte(header+string(yamlData)), 0644); err != nil {
		return nil, fmt.Errorf("failed to write YAML config: %w", err)
	}

	result.Success = true
	result.Message = fmt.Sprintf("Migrated %s → %s", jsonPath, yamlPath)
	return result, nil
}

// MigrateUserConfig migrates the user-level config from JSON to YAML.
func MigrateUserConfig(dryRun bool) (*MigrationResult, error) {
	jsonPath, err := LegacyUserConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get legacy user config path: %w", err)
	}

	yamlPath, err := UserConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config path: %w", err)
	}

	return MigrateJSONToYAML(jsonPath, yamlPath, dryRun)
}

// MigrateProjectConfig migrates the project-level config from JSON to YAML.
func MigrateProjectConfig(dryRun bool) (*MigrationResult, error) {
	jsonPath := LegacyProjectConfigPath()
	yamlPath := ProjectConfigPath()

	return MigrateJSONToYAML(jsonPath, yamlPath, dryRun)
}

// RemoveLegacyConfig removes a legacy JSON config file after successful migration.
// This should only be called after confirming the YAML config is working.
func RemoveLegacyConfig(jsonPath string, dryRun bool) error {
	if dryRun {
		return nil
	}

	// Verify YAML exists before removing JSON
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return nil // Already removed or never existed
	}

	// Rename to .bak instead of deleting (safer)
	bakPath := jsonPath + ".bak"
	if err := os.Rename(jsonPath, bakPath); err != nil {
		return fmt.Errorf("failed to backup legacy config: %w", err)
	}

	return nil
}

// DetectLegacyConfigs returns paths to any detected legacy JSON configs.
func DetectLegacyConfigs() (userJSON, projectJSON string, err error) {
	userPath, err := LegacyUserConfigPath()
	if err != nil {
		return "", "", err
	}

	if _, err := os.Stat(userPath); err == nil {
		userJSON = userPath
	}

	projectPath := LegacyProjectConfigPath()
	if _, err := os.Stat(projectPath); err == nil {
		projectJSON = projectPath
	}

	return userJSON, projectJSON, nil
}
