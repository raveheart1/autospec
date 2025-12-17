package notify

import (
	"errors"
	"sync"
)

// MockSender is a comprehensive mock implementation of Sender for testing.
// It records all method calls and allows configuring return values and errors.
type MockSender struct {
	mu sync.Mutex

	// Configuration
	VisualError     error
	SoundError      error
	visualAvailable bool
	soundAvailable  bool
	VisualFunc      func(Notification) error
	SoundFunc       func(string) error

	// Call tracking
	VisualCalls      []Notification
	SoundCalls       []string
	VisualCallCount  int
	SoundCallCount   int
	LastNotification Notification
	LastSoundFile    string
}

// NewMockSender creates a new mock sender with default behavior (all available, no errors)
func NewMockSender() *MockSender {
	return &MockSender{
		visualAvailable: true,
		soundAvailable:  true,
		VisualCalls:     make([]Notification, 0),
		SoundCalls:      make([]string, 0),
	}
}

// WithVisualError configures the mock to return an error on SendVisual
func (m *MockSender) WithVisualError(err error) *MockSender {
	m.VisualError = err
	return m
}

// WithSoundError configures the mock to return an error on SendSound
func (m *MockSender) WithSoundError(err error) *MockSender {
	m.SoundError = err
	return m
}

// WithVisualAvailable configures whether visual notifications are available
func (m *MockSender) WithVisualAvailable(available bool) *MockSender {
	m.visualAvailable = available
	return m
}

// WithSoundAvailable configures whether sound notifications are available
func (m *MockSender) WithSoundAvailable(available bool) *MockSender {
	m.soundAvailable = available
	return m
}

// WithVisualFunc configures a custom visual notification function
func (m *MockSender) WithVisualFunc(fn func(Notification) error) *MockSender {
	m.VisualFunc = fn
	return m
}

// WithSoundFunc configures a custom sound notification function
func (m *MockSender) WithSoundFunc(fn func(string) error) *MockSender {
	m.SoundFunc = fn
	return m
}

// SendVisual records the call and returns configured error
func (m *MockSender) SendVisual(n Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.VisualCalls = append(m.VisualCalls, n)
	m.VisualCallCount++
	m.LastNotification = n

	if m.VisualFunc != nil {
		return m.VisualFunc(n)
	}

	return m.VisualError
}

// SendSound records the call and returns configured error
func (m *MockSender) SendSound(soundFile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SoundCalls = append(m.SoundCalls, soundFile)
	m.SoundCallCount++
	m.LastSoundFile = soundFile

	if m.SoundFunc != nil {
		return m.SoundFunc(soundFile)
	}

	return m.SoundError
}

// VisualAvailable returns whether visual notifications are available
func (m *MockSender) VisualAvailable() bool {
	return m.visualAvailable
}

// SoundAvailable returns whether sound notifications are available
func (m *MockSender) SoundAvailable() bool {
	return m.soundAvailable
}

// Reset clears all recorded calls
func (m *MockSender) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.VisualCalls = make([]Notification, 0)
	m.SoundCalls = make([]string, 0)
	m.VisualCallCount = 0
	m.SoundCallCount = 0
	m.LastNotification = Notification{}
	m.LastSoundFile = ""
}

// AssertVisualCalled checks if SendVisual was called
func (m *MockSender) AssertVisualCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.VisualCallCount > 0
}

// AssertSoundCalled checks if SendSound was called
func (m *MockSender) AssertSoundCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.SoundCallCount > 0
}

// AssertVisualCalledWith checks if SendVisual was called with specific notification type
func (m *MockSender) AssertVisualCalledWith(notifType NotificationType) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, n := range m.VisualCalls {
		if n.NotificationType == notifType {
			return true
		}
	}
	return false
}

// AssertSoundCalledWith checks if SendSound was called with specific file
func (m *MockSender) AssertSoundCalledWith(soundFile string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, f := range m.SoundCalls {
		if f == soundFile {
			return true
		}
	}
	return false
}

// Common test errors
var (
	ErrMockVisual = errors.New("mock visual notification error")
	ErrMockSound  = errors.New("mock sound notification error")
)
