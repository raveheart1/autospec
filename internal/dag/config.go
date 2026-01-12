package dag

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ariel-frischer/autospec/internal/worktree"
)

// DAGExecutionConfig holds DAG-specific execution settings.
// These settings control behavior during dag run execution.
type DAGExecutionConfig struct {
	// OnConflict specifies default merge conflict handling.
	// Valid values: "manual" (default), "agent"
	OnConflict string `yaml:"on_conflict,omitempty" koanf:"on_conflict"`
	// BaseBranch is the default target branch for merging completed specs.
	// If empty, defaults to the repository's default branch (usually "main" or "master").
	BaseBranch string `yaml:"base_branch,omitempty" koanf:"base_branch"`
	// MaxSpecRetries is the max auto-retry attempts per spec.
	// 0 means manual retry only (default).
	MaxSpecRetries int `yaml:"max_spec_retries,omitempty" koanf:"max_spec_retries"`
	// MaxLogSize is the max log file size per spec (e.g., "50MB").
	// Default: "50MB"
	MaxLogSize string `yaml:"max_log_size,omitempty" koanf:"max_log_size"`
	// LogDir overrides the default log directory location.
	// If empty, defaults to XDG cache (~/.cache/autospec/dag-logs/).
	// Can also be set via AUTOSPEC_DAG_LOG_DIR environment variable.
	LogDir string `yaml:"log_dir,omitempty" koanf:"log_dir"`
}

// DefaultDAGConfig returns a DAGExecutionConfig with default values.
func DefaultDAGConfig() *DAGExecutionConfig {
	return &DAGExecutionConfig{
		OnConflict:     "manual",
		MaxSpecRetries: 0,
		MaxLogSize:     "50MB",
	}
}

// LoadDAGConfig loads DAG execution configuration with hierarchy:
// environment variables > provided config > defaults.
func LoadDAGConfig(cfg *DAGExecutionConfig) *DAGExecutionConfig {
	result := DefaultDAGConfig()

	// Apply provided config if any
	if cfg != nil {
		if cfg.OnConflict != "" {
			result.OnConflict = cfg.OnConflict
		}
		if cfg.BaseBranch != "" {
			result.BaseBranch = cfg.BaseBranch
		}
		if cfg.MaxSpecRetries > 0 {
			result.MaxSpecRetries = cfg.MaxSpecRetries
		}
		if cfg.MaxLogSize != "" {
			result.MaxLogSize = cfg.MaxLogSize
		}
		if cfg.LogDir != "" {
			result.LogDir = cfg.LogDir
		}
	}

	// Environment variables override everything
	result.applyEnvOverrides()

	return result
}

// applyEnvOverrides applies environment variable overrides to the config.
func (c *DAGExecutionConfig) applyEnvOverrides() {
	if val := os.Getenv("AUTOSPEC_DAG_ON_CONFLICT"); val != "" {
		c.OnConflict = val
	}
	if val := os.Getenv("AUTOSPEC_DAG_BASE_BRANCH"); val != "" {
		c.BaseBranch = val
	}
	if val := os.Getenv("AUTOSPEC_DAG_MAX_SPEC_RETRIES"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			c.MaxSpecRetries = n
		}
	}
	if val := os.Getenv("AUTOSPEC_DAG_MAX_LOG_SIZE"); val != "" {
		c.MaxLogSize = val
	}
	if val := os.Getenv("AUTOSPEC_DAG_LOG_DIR"); val != "" {
		c.LogDir = val
	}
}

// LoadWorktreeConfig loads worktree configuration with hierarchy:
// environment variables > provided config > defaults.
// This is a convenience wrapper around worktree.DefaultConfig with env overrides.
func LoadWorktreeConfig(cfg *worktree.WorktreeConfig) *worktree.WorktreeConfig {
	result := worktree.DefaultConfig()

	// Apply provided config if any
	if cfg != nil {
		if cfg.BaseDir != "" {
			result.BaseDir = cfg.BaseDir
		}
		if cfg.Prefix != "" {
			result.Prefix = cfg.Prefix
		}
		if cfg.SetupScript != "" {
			result.SetupScript = cfg.SetupScript
		}
		result.AutoSetup = cfg.AutoSetup
		result.TrackStatus = cfg.TrackStatus
		if len(cfg.CopyDirs) > 0 {
			result.CopyDirs = cfg.CopyDirs
		}
	}

	// Environment variables override everything
	applyWorktreeEnvOverrides(result)

	return result
}

// applyWorktreeEnvOverrides applies environment variable overrides.
func applyWorktreeEnvOverrides(c *worktree.WorktreeConfig) {
	if val := os.Getenv("AUTOSPEC_WORKTREE_BASE_DIR"); val != "" {
		c.BaseDir = val
	}
	if val := os.Getenv("AUTOSPEC_WORKTREE_PREFIX"); val != "" {
		c.Prefix = val
	}
	if val := os.Getenv("AUTOSPEC_WORKTREE_SETUP_SCRIPT"); val != "" {
		c.SetupScript = val
	}
	if val := os.Getenv("AUTOSPEC_WORKTREE_AUTO_SETUP"); val != "" {
		c.AutoSetup = val == "true" || val == "1"
	}
	if val := os.Getenv("AUTOSPEC_WORKTREE_TRACK_STATUS"); val != "" {
		c.TrackStatus = val == "true" || val == "1"
	}
}

// sizePattern matches size strings like "50MB", "100MB", "1GB", etc.
var sizePattern = regexp.MustCompile(`^(\d+)\s*(B|KB|MB|GB|TB)$`)

// ParseSize parses a human-readable size string (e.g., "50MB") into bytes.
// Supported units: B, KB, MB, GB, TB (case-insensitive).
// Returns the size in bytes or an error if the format is invalid.
func ParseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}

	matches := sizePattern.FindStringSubmatch(sizeStr)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid size format: %q (use format like 50MB, 100MB)", sizeStr)
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing size value: %w", err)
	}

	return applyMultiplier(value, matches[2])
}

// applyMultiplier converts value to bytes based on unit.
func applyMultiplier(value int64, unit string) (int64, error) {
	multipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}

	mult, ok := multipliers[unit]
	if !ok {
		return 0, fmt.Errorf("unknown size unit: %s", unit)
	}

	return value * mult, nil
}

// MaxLogSizeBytes returns the max log size in bytes from the config.
// Falls back to default (50MB) if not set or invalid.
func (c *DAGExecutionConfig) MaxLogSizeBytes() int64 {
	if c.MaxLogSize == "" {
		return 50 * 1024 * 1024 // 50MB default
	}

	bytes, err := ParseSize(c.MaxLogSize)
	if err != nil {
		return 50 * 1024 * 1024 // 50MB default on error
	}

	return bytes
}
