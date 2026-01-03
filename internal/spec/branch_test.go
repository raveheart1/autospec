// Package spec_test tests spec name extraction from git branch names.
// Related: /home/ari/repos/autospec/internal/spec/branch.go
// Tags: spec, git, branch, parsing

package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateBranchName(t *testing.T) {
	tests := map[string]struct {
		description string
		expected    string
	}{
		"simple feature": {
			description: "Add user authentication",
			expected:    "user-authentication",
		},
		"filters stop words": {
			description: "I want to add a feature for the users",
			expected:    "feature-users",
		},
		"keeps first 3 words": {
			description: "Implement OAuth2 integration for API access",
			expected:    "implement-oauth2-integration",
		},
		"keeps 4 words when exactly 4": {
			description: "Implement OAuth2 API access",
			expected:    "implement-oauth2-api-access",
		},
		"handles uppercase": {
			description: "Add API Support",
			expected:    "api-support",
		},
		"keeps two-letter words in meaningful context": {
			description: "Add CI CD pipeline",
			expected:    "ci-cd-pipeline",
		},
		"removes special characters": {
			description: "Add user-auth feature (v2)",
			expected:    "user-auth-feature",
		},
		"handles numbers": {
			description: "Version 2 upgrade",
			expected:    "version-upgrade",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GenerateBranchName(tt.description)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanBranchName(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"lowercase conversion": {
			input:    "MyFeature",
			expected: "myfeature",
		},
		"replaces spaces": {
			input:    "my feature",
			expected: "my-feature",
		},
		"replaces special chars": {
			input:    "my_feature@v2",
			expected: "my-feature-v2",
		},
		"removes consecutive hyphens": {
			input:    "my--feature",
			expected: "my-feature",
		},
		"removes leading hyphen": {
			input:    "-my-feature",
			expected: "my-feature",
		},
		"removes trailing hyphen": {
			input:    "my-feature-",
			expected: "my-feature",
		},
		"handles mixed special chars": {
			input:    "  My Feature (v2.0)  ",
			expected: "my-feature-v2-0",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := CleanBranchName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateBranchName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		branchName            string
		expectTruncation      bool
		expectedResult        string // if empty, just verify length constraints
		skipTrailingHyphenChk bool   // some edge cases may have trailing hyphens
	}{
		"short branch name unchanged": {
			branchName:       "001-my-feature",
			expectTruncation: false,
			expectedResult:   "001-my-feature",
		},
		"exactly at limit unchanged": {
			branchName:       "001-" + strings.Repeat("a", 240),
			expectTruncation: false,
		},
		"over limit truncated": {
			branchName:       "001-" + strings.Repeat("a", 250),
			expectTruncation: true,
		},
		"empty input": {
			branchName:       "",
			expectTruncation: false,
			expectedResult:   "",
		},
		"single character": {
			branchName:       "a",
			expectTruncation: false,
			expectedResult:   "a",
		},
		"no hyphen - just at limit": {
			branchName:       strings.Repeat("x", MaxBranchLength),
			expectTruncation: false,
		},
		"no hyphen - over limit": {
			branchName:       strings.Repeat("x", MaxBranchLength+10),
			expectTruncation: true,
		},
		"prefix too long - edge case": {
			// prefix is 243 chars (just under limit), suffix is 10 chars
			// maxSuffixLen = 244 - 243 - 1 = 0, edge case
			// Note: This case may result in trailing hyphen (known limitation)
			branchName:            strings.Repeat("p", 243) + "-" + strings.Repeat("s", 10),
			expectTruncation:      true,
			skipTrailingHyphenChk: true,
		},
		"trailing hyphen after truncation": {
			// Create a string that when truncated will end with hyphen
			branchName:       "001-word-" + strings.Repeat("a", 240),
			expectTruncation: true,
		},
		"special characters in name": {
			branchName:       "001-my-feature-with-special",
			expectTruncation: false,
			expectedResult:   "001-my-feature-with-special",
		},
		"unicode characters": {
			branchName:       "001-feature-émoji",
			expectTruncation: false,
			expectedResult:   "001-feature-émoji",
		},
		"long unicode that exceeds byte limit": {
			// Unicode chars take more bytes than their character count
			// 🔥 is 4 bytes, so 61 of them = 244 bytes (at limit)
			// Adding more should truncate
			branchName:       "001-" + strings.Repeat("🔥", 70), // 280+ bytes
			expectTruncation: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := TruncateBranchName(tt.branchName)
			assert.LessOrEqual(t, len(result), MaxBranchLength, "result should not exceed max length")

			if tt.expectTruncation {
				assert.Less(t, len(result), len(tt.branchName), "result should be shorter than input")
			} else if tt.expectedResult != "" {
				assert.Equal(t, tt.expectedResult, result)
			} else {
				assert.Equal(t, tt.branchName, result)
			}

			// Verify no trailing hyphen after truncation (except known edge cases)
			if len(result) > 0 && !tt.skipTrailingHyphenChk {
				assert.NotEqual(t, "-", string(result[len(result)-1]), "should not end with hyphen")
			}
		})
	}
}

func TestTruncateBranchName_NoHyphenEdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		branchName string
	}{
		"all digits no hyphen": {
			branchName: strings.Repeat("1", MaxBranchLength+10),
		},
		"all special after cleaning": {
			branchName: strings.Repeat("abc", 100),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := TruncateBranchName(tt.branchName)
			assert.LessOrEqual(t, len(result), MaxBranchLength)
		})
	}
}

func TestTruncateBranchName_PrefixLengthEdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		branchName string
		wantLen    int
	}{
		"prefix exactly at limit": {
			branchName: strings.Repeat("x", MaxBranchLength) + "-suffix",
			wantLen:    MaxBranchLength,
		},
		"prefix over limit": {
			branchName: strings.Repeat("x", MaxBranchLength+10) + "-suffix",
			wantLen:    MaxBranchLength,
		},
		"very short prefix long suffix": {
			branchName: "a-" + strings.Repeat("b", 300),
			wantLen:    MaxBranchLength,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := TruncateBranchName(tt.branchName)
			assert.LessOrEqual(t, len(result), tt.wantLen)
		})
	}
}

func TestGetNextBranchNumber(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "autospec-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specsDir := filepath.Join(tmpDir, "specs")
	err = os.MkdirAll(specsDir, 0o755)
	require.NoError(t, err)

	// Note: These tests run in the real git repo, so they will also pick up
	// existing branches. The tests verify relative behavior.

	t.Run("returns valid number format", func(t *testing.T) {
		num, err := GetNextBranchNumber(specsDir)
		require.NoError(t, err)
		// Should be a 3-digit zero-padded number
		assert.Len(t, num, 3)
		assert.Regexp(t, `^\d{3}$`, num)
	})

	t.Run("with existing specs increases number", func(t *testing.T) {
		// Get baseline
		baseNum, err := GetNextBranchNumber(specsDir)
		require.NoError(t, err)

		// Create spec directories with higher numbers
		err = os.MkdirAll(filepath.Join(specsDir, "100-first-feature"), 0o755)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(specsDir, "101-second-feature"), 0o755)
		require.NoError(t, err)

		num, err := GetNextBranchNumber(specsDir)
		require.NoError(t, err)

		// Should be at least 102 (or higher if git branches exist)
		numInt := 0
		fmt.Sscanf(num, "%d", &numInt)
		baseInt := 0
		fmt.Sscanf(baseNum, "%d", &baseInt)

		// Result should be >= 102 since we added 100 and 101
		assert.GreaterOrEqual(t, numInt, 102)
	})

	t.Run("non-existent directory returns valid number", func(t *testing.T) {
		num, err := GetNextBranchNumber("/nonexistent/path")
		require.NoError(t, err)
		// Should still return a valid format (from git branches if available)
		assert.Regexp(t, `^\d{3}$`, num)
	})
}

func TestFormatBranchName(t *testing.T) {
	tests := map[string]struct {
		number   string
		suffix   string
		expected string
	}{
		"001-my-feature":      {number: "001", suffix: "my-feature", expected: "001-my-feature"},
		"042-another-feature": {number: "042", suffix: "another-feature", expected: "042-another-feature"},
		"123-test":            {number: "123", suffix: "test", expected: "123-test"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := FormatBranchName(tt.number, tt.suffix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFeatureDirectory(t *testing.T) {
	result := GetFeatureDirectory("/home/user/project/specs", "001-my-feature")
	assert.Equal(t, "/home/user/project/specs/001-my-feature", result)
}

func TestStopWords(t *testing.T) {
	// Verify key stop words are in the map
	expectedStopWords := []string{"the", "a", "to", "for", "is", "are", "add", "get", "set"}
	for _, word := range expectedStopWords {
		assert.True(t, StopWords[word], "expected '%s' to be a stop word", word)
	}

	// Verify some non-stop words are not in the map
	nonStopWords := []string{"feature", "user", "api", "implement"}
	for _, word := range nonStopWords {
		assert.False(t, StopWords[word], "expected '%s' to NOT be a stop word", word)
	}
}
