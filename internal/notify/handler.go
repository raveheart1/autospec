package notify

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/term"
)

// Handler manages notification dispatch based on configuration and hooks.
// It wraps a Sender with configuration and provides hook methods for
// command completion, stage completion, and error notifications.
type Handler struct {
	config    NotificationConfig
	sender    Sender
	startTime time.Time
}

// NewHandler creates a new notification handler with the given configuration.
// The handler initializes with the current time as the command start time.
// If notifications are disabled in config, the handler will no-op on all calls.
func NewHandler(config NotificationConfig) *Handler {
	return &Handler{
		config:    config,
		sender:    NewSender(),
		startTime: time.Now(),
	}
}

// NewHandlerWithSender creates a handler with a custom sender (for testing).
func NewHandlerWithSender(config NotificationConfig, sender Sender) *Handler {
	return &Handler{
		config:    config,
		sender:    sender,
		startTime: time.Now(),
	}
}

// SetStartTime updates the command start time (useful for accurate duration tracking)
func (h *Handler) SetStartTime(t time.Time) {
	h.startTime = t
}

// Config returns the handler's notification configuration
func (h *Handler) Config() NotificationConfig {
	return h.config
}

// isEnabled checks if notifications should be sent.
// Returns false if notifications are disabled, running in CI, or non-interactive.
// Logs debug messages when notifications are skipped.
func (h *Handler) isEnabled() bool {
	log.Printf("[notify] debug: isEnabled() called - config.Enabled=%v", h.config.Enabled)
	if !h.config.Enabled {
		log.Printf("[notify] debug: notifications skipped - disabled in config")
		return false
	}

	// Check CI environment - auto-disable unless running interactively
	if isCI() {
		log.Printf("[notify] debug: notifications skipped - running in CI environment")
		return false
	}

	// Check TTY availability for interactive mode
	interactive := isInteractive()
	log.Printf("[notify] debug: isInteractive=%v (stdout=%v, stderr=%v, stdin=%v)",
		interactive,
		term.IsTerminal(int(os.Stdout.Fd())),
		term.IsTerminal(int(os.Stderr.Fd())),
		term.IsTerminal(int(os.Stdin.Fd())))
	if !interactive {
		log.Printf("[notify] debug: notifications skipped - non-interactive session (no TTY)")
		return false
	}

	log.Printf("[notify] debug: isEnabled() returning true")
	return true
}

// isCI checks for common CI environment variables.
// Returns true if any CI-related environment variable is set.
func isCI() bool {
	ciVars := []string{
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"TRAVIS",
		"JENKINS_URL",
		"BUILDKITE",
		"DRONE",
		"TEAMCITY_VERSION",
		"TF_BUILD",            // Azure DevOps
		"BITBUCKET_PIPELINES", // Bitbucket
		"CODEBUILD_BUILD_ID",  // AWS CodeBuild
		"HEROKU_TEST_RUN_ID",  // Heroku CI
		"NETLIFY",             // Netlify
		"VERCEL",              // Vercel
		"RENDER",              // Render
		"RAILWAY_ENVIRONMENT", // Railway
	}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// isInteractive checks if the session is interactive (has TTY).
// Checks stdout rather than stdin because CLI tools often have stdin piped
// while stdout remains connected to the terminal.
func isInteractive() bool {
	// Check stdout first (most reliable for CLI tools)
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return true
	}
	// Fall back to stderr (also commonly connected to terminal)
	if term.IsTerminal(int(os.Stderr.Fd())) {
		return true
	}
	// Finally check stdin
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// dispatch sends a notification asynchronously with a timeout.
// It respects the configured notification type (sound, visual, or both).
// Notification failures are logged but do not block command execution.
// Timeout is set to 5 seconds to allow audio files to play completely.
func (h *Handler) dispatch(n Notification) {
	log.Printf("[notify] debug: dispatch called - title=%s message=%s notificationType=%v", n.Title, n.Message, n.NotificationType)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.sendNotification(n)
	}()

	select {
	case <-done:
		log.Printf("[notify] debug: dispatch - notification sent successfully")
	case <-ctx.Done():
		log.Printf("[notify] debug: dispatch - timeout after 5s")
	}
}

// sendNotification sends the notification based on configured type
func (h *Handler) sendNotification(n Notification) {
	log.Printf("[notify] debug: sendNotification called - type=%v soundFile=%s", h.config.Type, h.config.SoundFile)
	switch h.config.Type {
	case OutputSound:
		log.Printf("[notify] debug: sending sound only")
		err := h.sender.SendSound(h.config.SoundFile)
		if err != nil {
			log.Printf("[notify] debug: SendSound error: %v", err)
		}
	case OutputVisual:
		log.Printf("[notify] debug: sending visual only")
		err := h.sender.SendVisual(n)
		if err != nil {
			log.Printf("[notify] debug: SendVisual error: %v", err)
		}
	case OutputBoth:
		log.Printf("[notify] debug: sending both visual and sound")
		err := h.sender.SendVisual(n)
		if err != nil {
			log.Printf("[notify] debug: SendVisual error: %v", err)
		}
		err = h.sender.SendSound(h.config.SoundFile)
		if err != nil {
			log.Printf("[notify] debug: SendSound error: %v", err)
		}
	default:
		log.Printf("[notify] debug: unknown notification type: %v", h.config.Type)
	}
}

// OnCommandComplete is called when an autospec command finishes.
// It sends a notification if the on_command_complete hook is enabled.
// If on_long_running is enabled, it only notifies if duration >= threshold.
func (h *Handler) OnCommandComplete(commandName string, success bool, duration time.Duration) {
	log.Printf("[notify] debug: OnCommandComplete called - command=%s success=%v duration=%v", commandName, success, duration)
	log.Printf("[notify] debug: config - OnCommandComplete=%v OnLongRunning=%v LongRunningThreshold=%v Type=%v SoundFile=%s",
		h.config.OnCommandComplete, h.config.OnLongRunning, h.config.LongRunningThreshold, h.config.Type, h.config.SoundFile)

	if !h.isEnabled() {
		log.Printf("[notify] debug: OnCommandComplete - skipped (not enabled)")
		return
	}

	// Check on_long_running first - if enabled and duration is below threshold, skip
	if h.config.OnLongRunning {
		threshold := h.config.LongRunningThreshold
		// 0 or negative threshold means "always notify"
		if threshold > 0 && duration < threshold {
			log.Printf("[notify] debug: OnCommandComplete - skipped (duration %v < threshold %v)", duration, threshold)
			return
		}
	}

	// Only notify if on_command_complete is enabled
	if !h.config.OnCommandComplete {
		log.Printf("[notify] debug: OnCommandComplete - skipped (on_command_complete=false)")
		return
	}

	notifType := TypeSuccess
	status := "completed successfully"
	if !success {
		notifType = TypeFailure
		status = "failed"
	}

	n := NewNotification(
		"autospec",
		fmt.Sprintf("Command '%s' %s (%s)", commandName, status, formatDuration(duration)),
		notifType,
	)
	log.Printf("[notify] debug: OnCommandComplete - calling dispatch")
	h.dispatch(n)
}

// OnStageComplete is called when a workflow stage finishes.
// It sends a notification if the on_stage_complete hook is enabled.
func (h *Handler) OnStageComplete(stageName string, success bool) {
	if !h.isEnabled() {
		return
	}

	if !h.config.OnStageComplete {
		return
	}

	notifType := TypeSuccess
	status := "completed"
	if !success {
		notifType = TypeFailure
		status = "failed"
	}

	n := NewNotification(
		"autospec",
		fmt.Sprintf("Stage '%s' %s", stageName, status),
		notifType,
	)
	h.dispatch(n)
}

// OnError is called when a command or stage fails.
// It sends a notification if the on_error hook is enabled.
// This is separate from OnCommandComplete/OnStageComplete to allow
// error-only notifications without command/stage completion notifications.
func (h *Handler) OnError(commandName string, err error) {
	if !h.isEnabled() {
		return
	}

	if !h.config.OnError {
		return
	}

	errMsg := "unknown error"
	if err != nil {
		errMsg = err.Error()
	}

	n := NewNotification(
		"autospec",
		fmt.Sprintf("Error in '%s': %s", commandName, errMsg),
		TypeFailure,
	)
	h.dispatch(n)
}

// formatDuration formats a duration for display in notifications
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
