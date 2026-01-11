package dag

import (
	"testing"
)

func TestSlugifyURL(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"https github url": {
			input:    "https://github.com/user/repo.git",
			expected: "github-com-user-repo",
		},
		"ssh github url": {
			input:    "git@github.com:user/repo.git",
			expected: "github-com-user-repo",
		},
		"https url without .git suffix": {
			input:    "https://github.com/user/repo",
			expected: "github-com-user-repo",
		},
		"gitlab ssh url": {
			input:    "git@gitlab.com:org/subgroup/repo.git",
			expected: "gitlab-com-org-subgroup-repo",
		},
		"http url": {
			input:    "http://bitbucket.org/user/repo.git",
			expected: "bitbucket-org-user-repo",
		},
		"git protocol url": {
			input:    "git://github.com/user/repo.git",
			expected: "github-com-user-repo",
		},
		"ssh protocol url": {
			input:    "ssh://git@github.com/user/repo.git",
			expected: "git-github-com-user-repo",
		},
		"url with special characters": {
			input:    "https://github.com/user_name/my-awesome_repo.git",
			expected: "github-com-user-name-my-awesome-repo",
		},
		"url with multiple special chars": {
			input:    "https://github.com/user//repo.git",
			expected: "github-com-user-repo",
		},
		"self-hosted gitlab": {
			input:    "git@git.company.com:team/project.git",
			expected: "git-company-com-team-project",
		},
		"url with port": {
			input:    "https://github.com:443/user/repo.git",
			expected: "github-com-443-user-repo",
		},
		"empty url": {
			input:    "",
			expected: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := slugifyURL(tt.input)
			if result != tt.expected {
				t.Errorf("slugifyURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRemoveProtocol(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"https protocol": {
			input:    "https://github.com/user/repo",
			expected: "github.com/user/repo",
		},
		"http protocol": {
			input:    "http://github.com/user/repo",
			expected: "github.com/user/repo",
		},
		"git protocol": {
			input:    "git://github.com/user/repo",
			expected: "github.com/user/repo",
		},
		"ssh protocol": {
			input:    "ssh://git@github.com/user/repo",
			expected: "git@github.com/user/repo",
		},
		"git@ prefix": {
			input:    "git@github.com:user/repo",
			expected: "github.com:user/repo",
		},
		"no protocol": {
			input:    "github.com/user/repo",
			expected: "github.com/user/repo",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := removeProtocol(tt.input)
			if result != tt.expected {
				t.Errorf("removeProtocol(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetPathHash(t *testing.T) {
	// Test that getPathHash returns a 12-character string
	hash := getPathHash()

	if len(hash) != 12 {
		t.Errorf("getPathHash() returned length %d, want 12", len(hash))
	}

	// Verify it's a valid hex string
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("getPathHash() returned non-hex character: %c", c)
		}
	}

	// Verify consistency - calling twice should return the same value
	hash2 := getPathHash()
	if hash != hash2 {
		t.Errorf("getPathHash() is not consistent: %q != %q", hash, hash2)
	}
}

func TestGetProjectID(t *testing.T) {
	// This test verifies GetProjectID returns a non-empty string.
	// The actual value depends on the git remote (if available) or path hash.
	projectID := GetProjectID()

	if projectID == "" {
		t.Error("GetProjectID() returned empty string")
	}

	// Verify consistency
	projectID2 := GetProjectID()
	if projectID != projectID2 {
		t.Errorf("GetProjectID() is not consistent: %q != %q", projectID, projectID2)
	}
}
