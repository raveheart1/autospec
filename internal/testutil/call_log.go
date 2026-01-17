package testutil

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// CallLogEntry represents a single call record in YAML format.
// It wraps CallRecord for serialization, handling error and time formatting.
type CallLogEntry struct {
	Method    string            `yaml:"method"`
	Prompt    string            `yaml:"prompt,omitempty"`
	Args      []string          `yaml:"args,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	Timestamp string            `yaml:"timestamp"`
	Response  string            `yaml:"response,omitempty"`
	Error     string            `yaml:"error,omitempty"`
	ExitCode  int               `yaml:"exit_code"`
}

// CallLog wraps []CallLogEntry for YAML serialization.
type CallLog struct {
	Entries []CallLogEntry `yaml:"entries"`
}

// WriteCallLog writes a slice of CallRecords to a YAML file.
func WriteCallLog(path string, records []CallRecord) error {
	log := CallLog{
		Entries: make([]CallLogEntry, 0, len(records)),
	}

	for _, r := range records {
		entry := callRecordToEntry(r)
		log.Entries = append(log.Entries, entry)
	}

	data, err := yaml.Marshal(log)
	if err != nil {
		return fmt.Errorf("marshaling call log to YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing call log to %s: %w", path, err)
	}

	return nil
}

// callRecordToEntry converts a CallRecord to a CallLogEntry.
func callRecordToEntry(r CallRecord) CallLogEntry {
	entry := CallLogEntry{
		Method:    r.Method,
		Prompt:    r.Prompt,
		Args:      r.Args,
		Env:       r.Env,
		Timestamp: r.Timestamp.Format(time.RFC3339Nano),
		Response:  r.Response,
		ExitCode:  r.ExitCode,
	}

	if r.Error != nil {
		entry.Error = r.Error.Error()
	}

	return entry
}

// ReadCallLog reads a YAML call log file and returns CallLogEntries.
// Note: Error field is returned as string since we cannot reconstruct the original error type.
func ReadCallLog(path string) (*CallLog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading call log from %s: %w", path, err)
	}

	var log CallLog
	if err := yaml.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("unmarshaling call log YAML: %w", err)
	}

	return &log, nil
}

// ToCallRecords converts CallLog entries back to CallRecords.
// Note: Error field will be nil; use entry.Error string for error checking.
func (log *CallLog) ToCallRecords() ([]CallRecord, error) {
	records := make([]CallRecord, 0, len(log.Entries))

	for i, entry := range log.Entries {
		record, err := entryToCallRecord(entry)
		if err != nil {
			return nil, fmt.Errorf("converting entry %d: %w", i, err)
		}
		records = append(records, record)
	}

	return records, nil
}

// entryToCallRecord converts a CallLogEntry back to a CallRecord.
func entryToCallRecord(entry CallLogEntry) (CallRecord, error) {
	ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
	if err != nil {
		return CallRecord{}, fmt.Errorf("parsing timestamp %q: %w", entry.Timestamp, err)
	}

	record := CallRecord{
		Method:    entry.Method,
		Prompt:    entry.Prompt,
		Args:      entry.Args,
		Env:       entry.Env,
		Timestamp: ts,
		Response:  entry.Response,
		ExitCode:  entry.ExitCode,
	}

	// Note: We cannot reconstruct the original error type, so Error remains nil.
	// Callers should check entry.Error string if error information is needed.

	return record, nil
}

// HasError returns true if the entry has a non-empty error string.
func (e CallLogEntry) HasError() bool {
	return e.Error != ""
}
