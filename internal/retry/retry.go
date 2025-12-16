package retry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RetryState represents retry tracking for a specific spec and phase combination
type RetryState struct {
	SpecName    string    `json:"spec_name"`
	Phase       string    `json:"phase"`
	Count       int       `json:"count"`
	LastAttempt time.Time `json:"last_attempt"`
	MaxRetries  int       `json:"max_retries"`
}

// RetryStore contains all retry states persisted to disk
type RetryStore struct {
	Retries     map[string]*RetryState           `json:"retries"`
	PhaseStates map[string]*PhaseExecutionState  `json:"phase_states,omitempty"`
}

// PhaseExecutionState tracks progress through phased implementation
type PhaseExecutionState struct {
	SpecName         string    `json:"spec_name"`
	CurrentPhase     int       `json:"current_phase"`
	TotalPhases      int       `json:"total_phases"`
	CompletedPhases  []int     `json:"completed_phases"`
	LastPhaseAttempt time.Time `json:"last_phase_attempt"`
}

// LoadRetryState loads retry state from persistent storage
// Performance contract: <10ms
func LoadRetryState(stateDir, specName, phase string, maxRetries int) (*RetryState, error) {
	store, err := loadStore(stateDir)
	if err != nil {
		// If file doesn't exist, return new state
		return &RetryState{
			SpecName:   specName,
			Phase:      phase,
			Count:      0,
			MaxRetries: maxRetries,
		}, nil
	}

	key := fmt.Sprintf("%s:%s", specName, phase)
	if state, exists := store.Retries[key]; exists {
		// Update MaxRetries in case it changed in config
		state.MaxRetries = maxRetries
		return state, nil
	}

	// Return new state if not found
	return &RetryState{
		SpecName:   specName,
		Phase:      phase,
		Count:      0,
		MaxRetries: maxRetries,
	}, nil
}

// SaveRetryState saves retry state to persistent storage using atomic write
func SaveRetryState(stateDir string, state *RetryState) error {
	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Load existing store
	store, err := loadStore(stateDir)
	if err != nil {
		// Create new store if loading failed
		store = &RetryStore{
			Retries: make(map[string]*RetryState),
		}
	}

	// Update entry
	key := fmt.Sprintf("%s:%s", state.SpecName, state.Phase)
	store.Retries[key] = state

	// Marshal to JSON
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal retry state: %w", err)
	}

	// Write to temp file
	retryPath := filepath.Join(stateDir, "retry.json")
	tmpPath := retryPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, retryPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// CanRetry returns true if more retries are allowed
func (r *RetryState) CanRetry() bool {
	return r.Count < r.MaxRetries
}

// Increment increments the retry count and updates the timestamp
// Returns an error if max retries are exceeded
func (r *RetryState) Increment() error {
	if !r.CanRetry() {
		return &RetryExhaustedError{
			SpecName:   r.SpecName,
			Phase:      r.Phase,
			Count:      r.Count,
			MaxRetries: r.MaxRetries,
		}
	}
	r.Count++
	r.LastAttempt = time.Now()
	return nil
}

// Reset resets the retry count and clears the timestamp
func (r *RetryState) Reset() {
	r.Count = 0
	r.LastAttempt = time.Time{}
}

// IncrementRetryCount is a convenience function that loads, increments, and saves
func IncrementRetryCount(stateDir, specName, phase string, maxRetries int) (*RetryState, error) {
	state, err := LoadRetryState(stateDir, specName, phase, maxRetries)
	if err != nil {
		return nil, err
	}

	if err := state.Increment(); err != nil {
		return nil, err
	}

	if err := SaveRetryState(stateDir, state); err != nil {
		return nil, err
	}

	return state, nil
}

// ResetRetryCount is a convenience function that loads, resets, and saves
func ResetRetryCount(stateDir, specName, phase string) error {
	// Load with default maxRetries (it doesn't matter since we're resetting)
	state, err := LoadRetryState(stateDir, specName, phase, 3)
	if err != nil {
		// If loading fails, nothing to reset
		return nil
	}

	state.Reset()
	return SaveRetryState(stateDir, state)
}

// loadStore loads the retry store from disk
func loadStore(stateDir string) (*RetryStore, error) {
	retryPath := filepath.Join(stateDir, "retry.json")
	data, err := os.ReadFile(retryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read retry state: %w", err)
	}

	var store RetryStore
	if err := json.Unmarshal(data, &store); err != nil {
		// If JSON is corrupted, return error so we create a new store
		return nil, fmt.Errorf("failed to unmarshal retry state: %w", err)
	}

	if store.Retries == nil {
		store.Retries = make(map[string]*RetryState)
	}

	return &store, nil
}

// LoadPhaseState loads phase execution state from persistent storage
func LoadPhaseState(stateDir, specName string) (*PhaseExecutionState, error) {
	store, err := loadStore(stateDir)
	if err != nil {
		// If file doesn't exist, return nil (no existing state)
		return nil, nil
	}

	if store.PhaseStates == nil {
		return nil, nil
	}

	return store.PhaseStates[specName], nil
}

// SavePhaseState persists phase state atomically via temp file + rename
func SavePhaseState(stateDir string, state *PhaseExecutionState) error {
	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Load existing store
	store, err := loadStore(stateDir)
	if err != nil {
		// Create new store if loading failed
		store = &RetryStore{
			Retries:     make(map[string]*RetryState),
			PhaseStates: make(map[string]*PhaseExecutionState),
		}
	}

	// Ensure PhaseStates map is initialized
	if store.PhaseStates == nil {
		store.PhaseStates = make(map[string]*PhaseExecutionState)
	}

	// Update entry
	store.PhaseStates[state.SpecName] = state

	// Marshal to JSON
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal phase state: %w", err)
	}

	// Write to temp file
	retryPath := filepath.Join(stateDir, "retry.json")
	tmpPath := retryPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, retryPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// MarkPhaseComplete adds a phase number to the completed_phases list
// Updates are persisted immediately
func MarkPhaseComplete(stateDir, specName string, phaseNumber int) error {
	state, err := LoadPhaseState(stateDir, specName)
	if err != nil {
		return err
	}

	if state == nil {
		// Create new state if none exists
		state = &PhaseExecutionState{
			SpecName:        specName,
			CompletedPhases: []int{},
		}
	}

	// Check if phase is already marked complete
	for _, p := range state.CompletedPhases {
		if p == phaseNumber {
			return nil // Already complete
		}
	}

	// Add phase to completed list
	state.CompletedPhases = append(state.CompletedPhases, phaseNumber)
	state.LastPhaseAttempt = time.Now()

	return SavePhaseState(stateDir, state)
}

// ResetPhaseState clears all phase tracking for a spec
func ResetPhaseState(stateDir, specName string) error {
	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Load existing store
	store, err := loadStore(stateDir)
	if err != nil {
		// Nothing to reset if store doesn't exist
		return nil
	}

	if store.PhaseStates == nil {
		return nil // Nothing to reset
	}

	// Delete the spec's phase state
	delete(store.PhaseStates, specName)

	// Marshal to JSON
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal phase state: %w", err)
	}

	// Write to temp file
	retryPath := filepath.Join(stateDir, "retry.json")
	tmpPath := retryPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, retryPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// IsPhaseCompleted checks if a phase is in the completed phases list
func (s *PhaseExecutionState) IsPhaseCompleted(phaseNumber int) bool {
	for _, p := range s.CompletedPhases {
		if p == phaseNumber {
			return true
		}
	}
	return false
}

// RetryExhaustedError indicates retry limit has been reached
type RetryExhaustedError struct {
	SpecName   string
	Phase      string
	Count      int
	MaxRetries int
}

func (e *RetryExhaustedError) Error() string {
	return fmt.Sprintf("retry limit exhausted for %s:%s (%d/%d attempts)",
		e.SpecName, e.Phase, e.Count, e.MaxRetries)
}

// ExitCode returns the exit code for retry exhausted (2)
func (e *RetryExhaustedError) ExitCode() int {
	return 2
}
