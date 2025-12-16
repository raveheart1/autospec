package notify

import "time"

// NotificationType represents the type of notification event
type NotificationType string

const (
	// TypeSuccess indicates a successful operation
	TypeSuccess NotificationType = "success"
	// TypeFailure indicates a failed operation
	TypeFailure NotificationType = "failure"
	// TypeInfo indicates an informational notification
	TypeInfo NotificationType = "info"
)

// OutputType represents the notification output type
type OutputType string

const (
	// OutputSound sends only an audible notification
	OutputSound OutputType = "sound"
	// OutputVisual sends only a visual notification
	OutputVisual OutputType = "visual"
	// OutputBoth sends both sound and visual notifications
	OutputBoth OutputType = "both"
)

// ValidOutputType checks if the given string is a valid output type
func ValidOutputType(s string) bool {
	switch OutputType(s) {
	case OutputSound, OutputVisual, OutputBoth:
		return true
	default:
		return false
	}
}

// NotificationConfig holds user preferences for notification behavior.
// Configuration is loaded from the config hierarchy (env > project > user > defaults).
type NotificationConfig struct {
	// Enabled is the master switch for all notifications (default: false, opt-in)
	Enabled bool `yaml:"enabled" json:"enabled" mapstructure:"enabled"`

	// Type specifies the notification output type: sound, visual, or both (default: both)
	Type OutputType `yaml:"type" json:"type" mapstructure:"type"`

	// SoundFile is an optional custom sound file path
	SoundFile string `yaml:"sound_file" json:"sound_file" mapstructure:"sound_file"`

	// OnCommandComplete notifies when any command finishes (default: true when enabled)
	OnCommandComplete bool `yaml:"on_command_complete" json:"on_command_complete" mapstructure:"on_command_complete"`

	// OnStageComplete notifies after each workflow stage (default: false)
	OnStageComplete bool `yaml:"on_stage_complete" json:"on_stage_complete" mapstructure:"on_stage_complete"`

	// OnError notifies on command/stage failure (default: true when enabled)
	OnError bool `yaml:"on_error" json:"on_error" mapstructure:"on_error"`

	// OnLongRunning notifies only if duration exceeds threshold (default: false)
	OnLongRunning bool `yaml:"on_long_running" json:"on_long_running" mapstructure:"on_long_running"`

	// LongRunningThreshold is the threshold for on_long_running hook (default: 30s)
	// A value of 0 or negative means "always notify"
	LongRunningThreshold time.Duration `yaml:"long_running_threshold" json:"long_running_threshold" mapstructure:"long_running_threshold"`
}

// DefaultConfig returns a NotificationConfig with default values
func DefaultConfig() NotificationConfig {
	return NotificationConfig{
		Enabled:              false,
		Type:                 OutputBoth,
		SoundFile:            "",
		OnCommandComplete:    true,
		OnStageComplete:      false,
		OnError:              true,
		OnLongRunning:        false,
		LongRunningThreshold: 30 * time.Second,
	}
}

// Notification represents a single notification event to dispatch
type Notification struct {
	// Title is the notification title (e.g., "autospec")
	Title string

	// Message is the notification body text
	Message string

	// NotificationType indicates the event type: success, failure, or info
	NotificationType NotificationType
}

// NewNotification creates a new Notification with the given parameters
func NewNotification(title, message string, notificationType NotificationType) Notification {
	return Notification{
		Title:            title,
		Message:          message,
		NotificationType: notificationType,
	}
}
