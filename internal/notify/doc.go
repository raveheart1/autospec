// Package notify provides cross-platform notification support for autospec.
//
// The notify package implements a notification system that alerts users when
// autospec commands and workflow stages complete. It supports three major
// operating systems (macOS, Linux, Windows) using only os/exec to call native
// OS tools, ensuring zero external Go dependencies and CGO_ENABLED=0 compatibility.
//
// # Features
//
//   - Visual notifications via native OS notification systems
//   - Audio alerts via system sound tools
//   - Configurable notification hooks (on_command_complete, on_stage_complete, on_error, on_long_running)
//   - Graceful degradation when notification tools are unavailable
//   - Non-blocking async dispatch with configurable timeout
//
// # Platform Support
//
//   - macOS: osascript for visual notifications, afplay for sound
//   - Linux: notify-send for visual notifications, paplay for sound
//   - Windows: PowerShell for toast notifications and sound
//
// # Usage
//
//	config := notify.NotificationConfig{
//		Enabled:           true,
//		Type:              notify.TypeBoth,
//		OnCommandComplete: true,
//		OnError:           true,
//	}
//	handler := notify.NewHandler(config)
//	handler.OnCommandComplete("run", true, 5*time.Second)
package notify
