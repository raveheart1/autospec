package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "update", updateCmd.Use)
	assert.NotEmpty(t, updateCmd.Short)
	assert.NotEmpty(t, updateCmd.Long)
	assert.NotEmpty(t, updateCmd.Example)
	assert.NotNil(t, updateCmd.RunE)
}

func TestUpdateCmd_DevBuildPreventsUpdate(t *testing.T) {
	// Save and restore original version
	origVersion := Version
	Version = "dev"
	defer func() { Version = origVersion }()

	err := runUpdate(updateCmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dev builds")
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		bytes int64
		want  string
	}{
		"bytes": {
			bytes: 100,
			want:  "100 B",
		},
		"kilobytes": {
			bytes: 1024,
			want:  "1.0 KB",
		},
		"megabytes": {
			bytes: 1024 * 1024,
			want:  "1.0 MB",
		},
		"megabytes with decimal": {
			bytes: 1024*1024 + 512*1024,
			want:  "1.5 MB",
		},
		"gigabytes": {
			bytes: 1024 * 1024 * 1024,
			want:  "1.0 GB",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatBytes(tt.bytes))
		})
	}
}

func TestPrintProgress(t *testing.T) {
	t.Parallel()

	// Test that printProgress doesn't panic with various inputs
	tests := map[string]struct {
		current int64
		total   int64
	}{
		"zero total": {
			current: 100,
			total:   0,
		},
		"negative total": {
			current: 100,
			total:   -1,
		},
		"partial progress": {
			current: 50,
			total:   100,
		},
		"complete": {
			current: 100,
			total:   100,
		},
		"zero current": {
			current: 0,
			total:   100,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Just verify it doesn't panic
			printProgress(tt.current, tt.total)
		})
	}
}
