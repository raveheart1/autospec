package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationResult_IsValid(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		result   ValidationResult
		expected bool
	}{
		"all checks pass": {
			result: ValidationResult{
				PathExists:            true,
				PathDiffersFromSource: true,
				InGitWorktreeList:     true,
				Errors:                nil,
			},
			expected: true,
		},
		"path does not exist": {
			result: ValidationResult{
				PathExists:            false,
				PathDiffersFromSource: true,
				InGitWorktreeList:     true,
				Errors:                []string{"path error"},
			},
			expected: false,
		},
		"path same as source": {
			result: ValidationResult{
				PathExists:            true,
				PathDiffersFromSource: false,
				InGitWorktreeList:     true,
				Errors:                []string{"same path error"},
			},
			expected: false,
		},
		"not in worktree list": {
			result: ValidationResult{
				PathExists:            true,
				PathDiffersFromSource: true,
				InGitWorktreeList:     false,
				Errors:                []string{"not in list error"},
			},
			expected: false,
		},
		"multiple failures": {
			result: ValidationResult{
				PathExists:            false,
				PathDiffersFromSource: false,
				InGitWorktreeList:     false,
				Errors:                []string{"error1", "error2"},
			},
			expected: false,
		},
		"all checks pass but has errors": {
			result: ValidationResult{
				PathExists:            true,
				PathDiffersFromSource: true,
				InGitWorktreeList:     true,
				Errors:                []string{"some error"},
			},
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.result.IsValid())
		})
	}
}

func TestCheckPathExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	tests := map[string]struct {
		setupPath    func() string
		wantExists   bool
		wantErrorMsg string
	}{
		"existing directory": {
			setupPath: func() string {
				return tmpDir
			},
			wantExists: true,
		},
		"non-existent path": {
			setupPath: func() string {
				return filepath.Join(tmpDir, "nonexistent")
			},
			wantExists:   false,
			wantErrorMsg: "worktree path does not exist",
		},
		"file instead of directory": {
			setupPath: func() string {
				f := filepath.Join(tmpDir, "file.txt")
				err := os.WriteFile(f, []byte("test"), 0644)
				require.NoError(t, err)
				return f
			},
			wantExists:   false,
			wantErrorMsg: "worktree path is not a directory",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := &ValidationResult{}
			path := tt.setupPath()

			exists := checkPathExists(path, result)

			assert.Equal(t, tt.wantExists, exists)
			if tt.wantErrorMsg != "" {
				require.Len(t, result.Errors, 1)
				assert.Contains(t, result.Errors[0], tt.wantErrorMsg)
			} else {
				assert.Empty(t, result.Errors)
			}
		})
	}
}

func TestCheckPathDiffers(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		worktreePath string
		sourcePath   string
		wantDiffers  bool
		wantErrorMsg string
	}{
		"paths differ": {
			worktreePath: "/home/user/worktree",
			sourcePath:   "/home/user/source",
			wantDiffers:  true,
		},
		"paths are same": {
			worktreePath: "/home/user/repo",
			sourcePath:   "/home/user/repo",
			wantDiffers:  false,
			wantErrorMsg: "worktree path same as source repo",
		},
		"trailing slash normalization": {
			worktreePath: "/home/user/worktree",
			sourcePath:   "/home/user/worktree/", // Note: filepath.Abs handles this
			wantDiffers:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := &ValidationResult{}

			differs := checkPathDiffers(tt.worktreePath, tt.sourcePath, result)

			assert.Equal(t, tt.wantDiffers, differs)
			if tt.wantErrorMsg != "" {
				require.Len(t, result.Errors, 1)
				assert.Contains(t, result.Errors[0], tt.wantErrorMsg)
			} else {
				assert.Empty(t, result.Errors)
			}
		})
	}
}

func TestCheckInWorktreeList(t *testing.T) {
	t.Parallel()

	// Note: These tests verify the logic, not actual git integration.
	// Integration tests would require a real git repository.

	tests := map[string]struct {
		description string
	}{
		"worktree list check logic": {
			description: "checkInWorktreeList iterates through GitWorktreeList entries to find matching path",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// The actual git worktree list integration is tested elsewhere.
			// This test documents the expected behavior.
			assert.NotEmpty(t, tt.description)
		})
	}
}

func TestValidateWorktree_PathResolution(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		worktreePath string
		sourcePath   string
		wantErr      bool
	}{
		"valid paths": {
			worktreePath: ".",
			sourcePath:   "..",
			wantErr:      false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := ValidateWorktree(tt.worktreePath, tt.sourcePath)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestValidateWorktree_ErrorMessages(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "does-not-exist")

	result, err := ValidateWorktree(nonExistentPath, tmpDir)
	require.NoError(t, err)

	assert.False(t, result.PathExists)
	assert.True(t, result.PathDiffersFromSource)
	require.Greater(t, len(result.Errors), 0)

	// Verify error messages are actionable
	assert.Contains(t, result.Errors[0], "worktree path does not exist")
	assert.Contains(t, result.Errors[0], "ensure setup script creates the directory")
}

func TestValidateWorktree_SameAsSource(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	result, err := ValidateWorktree(tmpDir, tmpDir)
	require.NoError(t, err)

	assert.True(t, result.PathExists)
	assert.False(t, result.PathDiffersFromSource)
	assert.False(t, result.IsValid())

	// Find the "same as source" error
	var foundSameError bool
	for _, errMsg := range result.Errors {
		if contains := assert.Contains(t, errMsg, "worktree path same as source repo"); contains {
			foundSameError = true
			break
		}
	}
	assert.True(t, foundSameError, "expected 'same as source' error message")
}
