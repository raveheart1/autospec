package dag

import (
	"os"
	"strconv"

	"github.com/ariel-frischer/autospec/internal/worktree"
)

// DAGExecutionConfig holds DAG-specific execution settings.
// These settings control behavior during dag run execution.
type DAGExecutionConfig struct {
	// OnConflict specifies default merge conflict handling.
	// Valid values: "manual" (default), "agent"
	OnConflict string `yaml:"on_conflict,omitempty" koanf:"on_conflict"`
	// MaxSpecRetries is the max auto-retry attempts per spec.
	// 0 means manual retry only (default).
	MaxSpecRetries int `yaml:"max_spec_retries,omitempty" koanf:"max_spec_retries"`
	// MaxLogSize is the max log file size per spec (e.g., "50MB").
	// Default: "50MB"
	MaxLogSize string `yaml:"max_log_size,omitempty" koanf:"max_log_size"`
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
		if cfg.MaxSpecRetries > 0 {
			result.MaxSpecRetries = cfg.MaxSpecRetries
		}
		if cfg.MaxLogSize != "" {
			result.MaxLogSize = cfg.MaxLogSize
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
	if val := os.Getenv("AUTOSPEC_DAG_MAX_SPEC_RETRIES"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			c.MaxSpecRetries = n
		}
	}
	if val := os.Getenv("AUTOSPEC_DAG_MAX_LOG_SIZE"); val != "" {
		c.MaxLogSize = val
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
