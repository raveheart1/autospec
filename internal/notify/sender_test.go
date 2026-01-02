// Package notify_test tests notification sending via macOS notification center.
// Related: /home/ari/repos/autospec/internal/notify/sender.go
// Tags: notify, sender, macos, osascript

package notify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlatform(t *testing.T) {
	t.Parallel()
	platform := Platform()

	// Platform should return a non-empty string
	if platform == "" {
		t.Error("Platform() returned empty string")
	}

	// Platform should be one of the expected values
	validPlatforms := map[string]bool{
		"darwin":  true,
		"linux":   true,
		"windows": true,
		"freebsd": true,
		"openbsd": true,
		"netbsd":  true,
	}

	if !validPlatforms[platform] {
		// Not an error - could be running on uncommon platform
		t.Logf("Running on uncommon platform: %s", platform)
	}
}

func TestNewSender(t *testing.T) {
	t.Parallel()
	sender := NewSender()

	// Sender should never be nil
	if sender == nil {
		t.Fatal("NewSender() returned nil")
	}

	// Sender should implement the interface
	var _ Sender = sender
}

func TestSenderInterface(t *testing.T) {
	t.Parallel()
	sender := NewSender()

	// Test that interface is implemented and availability methods work
	// NOTE: We do NOT call SendVisual/SendSound to avoid triggering real OS notifications
	t.Run("VisualAvailable", func(t *testing.T) {
		// Should return bool without panic
		_ = sender.VisualAvailable()
	})

	t.Run("SoundAvailable", func(t *testing.T) {
		// Should return bool without panic
		_ = sender.SoundAvailable()
	})

	t.Run("implements Sender interface", func(t *testing.T) {
		// Verify the sender implements the interface at compile time
		var _ Sender = sender
	})
}

func TestNoopSender(t *testing.T) {
	t.Parallel()
	sender := &noopSender{}

	tests := map[string]struct {
		fn       func() interface{}
		expected interface{}
	}{
		"VisualAvailable returns false": {
			fn:       func() interface{} { return sender.VisualAvailable() },
			expected: false,
		},
		"SoundAvailable returns false": {
			fn:       func() interface{} { return sender.SoundAvailable() },
			expected: false,
		},
		"SendVisual returns nil": {
			fn: func() interface{} {
				return sender.SendVisual(NewNotification("test", "test", TypeInfo))
			},
			expected: nil,
		},
		"SendSound returns nil": {
			fn:       func() interface{} { return sender.SendSound("") },
			expected: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := tt.fn()
			if result != tt.expected {
				t.Errorf("got %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidateSoundFile(t *testing.T) {
	t.Parallel()
	// Create a temporary test file
	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "test.wav")
	if err := os.WriteFile(validFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a directory for testing
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	tests := map[string]struct {
		soundFile string
		expected  string
	}{
		"empty string returns empty": {
			soundFile: "",
			expected:  "",
		},
		"valid wav file returns path": {
			soundFile: validFile,
			expected:  validFile,
		},
		"non-existent file returns empty": {
			soundFile: "/path/to/nonexistent/file.wav",
			expected:  "",
		},
		"directory returns empty": {
			soundFile: testDir,
			expected:  "",
		},
		"unsupported extension returns empty": {
			soundFile: filepath.Join(tmpDir, "test.txt"),
			expected:  "",
		},
	}

	// Create test.txt for unsupported extension test
	unsupportedFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(unsupportedFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create unsupported test file: %v", err)
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := ValidateSoundFile(tt.soundFile)
			if result != tt.expected {
				t.Errorf("ValidateSoundFile(%q) = %q, expected %q", tt.soundFile, result, tt.expected)
			}
		})
	}
}

func TestValidateSoundFile_SupportedExtensions(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	tests := map[string]struct {
		extension string
		valid     bool
	}{
		".wav":       {extension: ".wav", valid: true},
		".mp3":       {extension: ".mp3", valid: true},
		".aiff":      {extension: ".aiff", valid: true},
		".aif":       {extension: ".aif", valid: true},
		".ogg":       {extension: ".ogg", valid: true},
		".flac":      {extension: ".flac", valid: true},
		".m4a":       {extension: ".m4a", valid: true},
		".txt":       {extension: ".txt", valid: false},
		".exe":       {extension: ".exe", valid: false},
		".png":       {extension: ".png", valid: false},
		".WAV upper": {extension: ".WAV", valid: true},  // case insensitive
		".MP3 upper": {extension: ".MP3", valid: true},  // case insensitive
		".Aiff":      {extension: ".Aiff", valid: true}, // mixed case
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create test file with the extension
			testFile := filepath.Join(tmpDir, "test"+tt.extension)
			if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			result := ValidateSoundFile(testFile)
			if tt.valid && result == "" {
				t.Errorf("expected valid file %q to return path, got empty", testFile)
			}
			if !tt.valid && result != "" {
				t.Errorf("expected invalid extension %q to return empty, got %q", tt.extension, result)
			}
		})
	}
}

func TestSupportedAudioExtensions(t *testing.T) {
	t.Parallel()
	// Verify the global map contains expected extensions
	expectedExtensions := []string{".wav", ".mp3", ".aiff", ".aif", ".ogg", ".flac", ".m4a"}

	for _, ext := range expectedExtensions {
		if !supportedAudioExtensions[ext] {
			t.Errorf("expected extension %q to be supported", ext)
		}
	}

	// Verify some unsupported extensions
	unsupportedExtensions := []string{".txt", ".exe", ".doc", ".pdf"}
	for _, ext := range unsupportedExtensions {
		if supportedAudioExtensions[ext] {
			t.Errorf("expected extension %q to be unsupported", ext)
		}
	}
}

func TestToolAvailable(t *testing.T) {
	t.Parallel()
	// Test with a command that should exist on most systems
	tests := map[string]struct {
		tool     string
		expected bool
	}{
		"should find common tool": {
			// Use different common tools based on context
			// At minimum, "go" should be available in Go test environment
			tool:     "go",
			expected: true,
		},
		"should not find nonexistent tool": {
			tool:     "nonexistent_tool_12345",
			expected: false,
		},
		"empty string returns false": {
			tool:     "",
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := toolAvailable(tt.tool)
			if result != tt.expected {
				t.Errorf("toolAvailable(%q) = %v, expected %v", tt.tool, result, tt.expected)
			}
		})
	}
}

// mockSender is a test implementation of Sender for verification
type mockSender struct {
	visualAvailable  bool
	soundAvailable   bool
	visualCalled     bool
	soundCalled      bool
	lastNotification Notification
	lastSoundFile    string
	sendVisualError  error
	sendSoundError   error
}

func (m *mockSender) SendVisual(n Notification) error {
	m.visualCalled = true
	m.lastNotification = n
	return m.sendVisualError
}

func (m *mockSender) SendSound(soundFile string) error {
	m.soundCalled = true
	m.lastSoundFile = soundFile
	return m.sendSoundError
}

func (m *mockSender) VisualAvailable() bool {
	return m.visualAvailable
}

func (m *mockSender) SoundAvailable() bool {
	return m.soundAvailable
}

func TestMockSender(t *testing.T) {
	t.Parallel()
	// Verify mock sender implements the interface correctly
	mock := &mockSender{
		visualAvailable: true,
		soundAvailable:  true,
	}

	var sender Sender = mock

	// Test interface methods
	if !sender.VisualAvailable() {
		t.Error("expected VisualAvailable to be true")
	}
	if !sender.SoundAvailable() {
		t.Error("expected SoundAvailable to be true")
	}

	n := NewNotification("test", "message", TypeSuccess)
	if err := sender.SendVisual(n); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !mock.visualCalled {
		t.Error("expected visualCalled to be true")
	}
	if mock.lastNotification != n {
		t.Error("notification not recorded correctly")
	}

	if err := sender.SendSound("/test/sound.wav"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !mock.soundCalled {
		t.Error("expected soundCalled to be true")
	}
	if mock.lastSoundFile != "/test/sound.wav" {
		t.Errorf("sound file not recorded correctly: got %q", mock.lastSoundFile)
	}
}
