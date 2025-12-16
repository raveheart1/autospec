// Package retry provides persistent retry state management for autospec workflows.
// It tracks retry attempts per spec:stage combination, stage execution progress for
// phased implementation, and task-level execution state. State is persisted to
// ~/.autospec/state/retry.json with atomic writes for concurrency safety.
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
	Retries     map[string]*RetryState          `json:"retries"`
	StageStates map[string]*StageExecutionState `json:"stage_states,omitempty"`
	TaskStates  map[string]*TaskExecutionState  `json:"task_states,omitempty"`
}

// retryStoreLegacy is used for backward-compatible loading of old retry state files
// that used "phase_states" instead of "stage_states"
type retryStoreLegacy struct {
	Retries     map[string]*RetryState          `json:"retries"`
	PhaseStates map[string]*StageExecutionState `json:"phase_states,omitempty"`
	StageStates map[string]*StageExecutionState `json:"stage_states,omitempty"`
	TaskStates  map[string]*TaskExecutionState  `json:"task_states,omitempty"`
}

// StageExecutionState tracks progress through phased implementation
type StageExecutionState struct {
	SpecName         string    `json:"spec_name"`
	CurrentPhase     int       `json:"current_phase"`
	TotalPhases      int       `json:"total_phases"`
	CompletedPhases  []int     `json:"completed_phases"`
	LastPhaseAttempt time.Time `json:"last_phase_attempt"`
}

// TaskExecutionState tracks progress through task-level execution mode
type TaskExecutionState struct {
	SpecName         string    `json:"spec_name"`
	CurrentTaskID    string    `json:"current_task_id"`
	CompletedTaskIDs []string  `json:"completed_task_ids"`
	TotalTasks       int       `json:"total_tasks"`
	LastTaskAttempt  time.Time `json:"last_task_attempt"`
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

// loadStore loads the retry store from disk with backward-compatible parsing
// for old retry state files that used "phase_states" instead of "stage_states"
func loadStore(stateDir string) (*RetryStore, error) {
	retryPath := filepath.Join(stateDir, "retry.json")
	data, err := os.ReadFile(retryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read retry state: %w", err)
	}

	// Use legacy struct to handle both old (phase_states) and new (stage_states) formats
	var legacy retryStoreLegacy
	if err := json.Unmarshal(data, &legacy); err != nil {
		// If JSON is corrupted, return error so we create a new store
		return nil, fmt.Errorf("failed to unmarshal retry state: %w", err)
	}

	// Create the current store
	store := &RetryStore{
		Retries:     legacy.Retries,
		StageStates: legacy.StageStates,
		TaskStates:  legacy.TaskStates,
	}

	if store.Retries == nil {
		store.Retries = make(map[string]*RetryState)
	}

	// Migrate legacy phase_states to stage_states if present
	if legacy.PhaseStates != nil && len(legacy.PhaseStates) > 0 {
		if store.StageStates == nil {
			store.StageStates = make(map[string]*StageExecutionState)
		}
		// Copy legacy phase states to stage states (migration)
		for key, state := range legacy.PhaseStates {
			// Only migrate if not already present in stage_states
			if _, exists := store.StageStates[key]; !exists {
				store.StageStates[key] = state
			}
		}
	}

	return store, nil
}

// LoadStageState loads stage execution state from persistent storage
func LoadStageState(stateDir, specName string) (*StageExecutionState, error) {
	store, err := loadStore(stateDir)
	if err != nil {
		// If file doesn't exist, return nil (no existing state)
		return nil, nil
	}

	if store.StageStates == nil {
		return nil, nil
	}

	return store.StageStates[specName], nil
}

// SaveStageState persists stage state atomically via temp file + rename
func SaveStageState(stateDir string, state *StageExecutionState) error {
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
			StageStates: make(map[string]*StageExecutionState),
		}
	}

	// Ensure StageStates map is initialized
	if store.StageStates == nil {
		store.StageStates = make(map[string]*StageExecutionState)
	}

	// Update entry
	store.StageStates[state.SpecName] = state

	// Marshal to JSON
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stage state: %w", err)
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

// MarkStageComplete adds a phase number to the completed_phases list
// Updates are persisted immediately
func MarkStageComplete(stateDir, specName string, phaseNumber int) error {
	state, err := LoadStageState(stateDir, specName)
	if err != nil {
		return fmt.Errorf("loading stage state: %w", err)
	}

	if state == nil {
		// Create new state if none exists
		state = &StageExecutionState{
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

	return SaveStageState(stateDir, state)
}

// ResetStageState clears all stage tracking for a spec
func ResetStageState(stateDir, specName string) error {
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

	if store.StageStates == nil {
		return nil // Nothing to reset
	}

	// Delete the spec's stage state
	delete(store.StageStates, specName)

	// Marshal to JSON
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stage state: %w", err)
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
func (s *StageExecutionState) IsPhaseCompleted(phaseNumber int) bool {
	for _, p := range s.CompletedPhases {
		if p == phaseNumber {
			return true
		}
	}
	return false
}

// LoadTaskState loads task execution state from persistent storage
func LoadTaskState(stateDir, specName string) (*TaskExecutionState, error) {
	store, err := loadStore(stateDir)
	if err != nil {
		// If file doesn't exist, return nil (no existing state)
		return nil, nil
	}

	if store.TaskStates == nil {
		return nil, nil
	}

	return store.TaskStates[specName], nil
}

// SaveTaskState persists task state atomically via temp file + rename
func SaveTaskState(stateDir string, state *TaskExecutionState) error {
	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Load existing store
	store, err := loadStore(stateDir)
	if err != nil {
		// Create new store if loading failed
		store = &RetryStore{
			Retries:    make(map[string]*RetryState),
			TaskStates: make(map[string]*TaskExecutionState),
		}
	}

	// Ensure TaskStates map is initialized
	if store.TaskStates == nil {
		store.TaskStates = make(map[string]*TaskExecutionState)
	}

	// Update entry
	store.TaskStates[state.SpecName] = state

	// Marshal to JSON
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task state: %w", err)
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

// MarkTaskComplete adds a task ID to the completed_task_ids list
// Updates are persisted immediately
func MarkTaskComplete(stateDir, specName, taskID string) error {
	state, err := LoadTaskState(stateDir, specName)
	if err != nil {
		return fmt.Errorf("loading task state: %w", err)
	}

	if state == nil {
		// Create new state if none exists
		state = &TaskExecutionState{
			SpecName:         specName,
			CompletedTaskIDs: []string{},
		}
	}

	// Check if task is already marked complete
	for _, t := range state.CompletedTaskIDs {
		if t == taskID {
			return nil // Already complete
		}
	}

	// Add task to completed list
	state.CompletedTaskIDs = append(state.CompletedTaskIDs, taskID)
	state.LastTaskAttempt = time.Now()

	return SaveTaskState(stateDir, state)
}

// ResetTaskState clears all task tracking for a spec
func ResetTaskState(stateDir, specName string) error {
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

	if store.TaskStates == nil {
		return nil // Nothing to reset
	}

	// Delete the spec's task state
	delete(store.TaskStates, specName)

	// Marshal to JSON
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task state: %w", err)
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

// IsTaskCompleted checks if a task ID is in the completed tasks list
func (s *TaskExecutionState) IsTaskCompleted(taskID string) bool {
	for _, t := range s.CompletedTaskIDs {
		if t == taskID {
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
