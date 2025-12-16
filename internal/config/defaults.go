package config

import "time"

// GetDefaults returns the default configuration values
func GetDefaults() map[string]interface{} {
	return map[string]interface{}{
		"claude_cmd": "claude",
		"claude_args": []string{
			"-p",
			"--verbose",
			"--output-format",
			"stream-json",
		},
		"use_api_key":        false,
		"custom_claude_cmd":  "",
		"max_retries":        3,
		"specs_dir":          "./specs",
		"state_dir":          "~/.autospec/state",
		"skip_preflight":     false,
		"timeout":            2400,  // 40 minutes default
		"show_progress":      false, // Progress indicators off by default (professional)
		"skip_confirmations": false, // Confirmation prompts enabled by default
		// implement_method: Default to "phases" for cost-efficient execution with context isolation.
		// This changes the legacy behavior (single-session) to run each phase in a separate Claude session.
		// Valid values: "single-session", "phases", "tasks"
		"implement_method": "phases",
		// notifications: Notification settings for command and stage completion.
		// Disabled by default (opt-in). When enabled, defaults to both sound and visual notifications.
		"notifications": map[string]interface{}{
			"enabled":                false,                       // Disabled by default (opt-in)
			"type":                   "both",                      // Both sound and visual when enabled
			"sound_file":             "",                          // Use system default sound
			"on_command_complete":    true,                        // Notify when command finishes (default when enabled)
			"on_stage_complete":      false,                       // Don't notify on each stage by default
			"on_error":               true,                        // Notify on failures (default when enabled)
			"on_long_running":        false,                       // Don't use duration threshold by default
			"long_running_threshold": (30 * time.Second).String(), // 30 seconds threshold
		},
	}
}
