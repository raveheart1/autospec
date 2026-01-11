package dag

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GetProjectID returns a unique identifier for the current project.
// The ID is derived from the git remote origin URL when available,
// or from a hash of the absolute path for local-only repositories.
//
// The returned ID is used to organize logs in the user cache directory,
// ensuring logs from different projects are stored separately.
func GetProjectID() string {
	// Try to get git remote origin URL
	remoteURL := getGitRemoteURL()
	if remoteURL != "" {
		return slugifyURL(remoteURL)
	}

	// Fallback to path hash for local-only repositories
	return getPathHash()
}

// getGitRemoteURL returns the git remote origin URL, or empty string if unavailable.
func getGitRemoteURL() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getPathHash returns the first 12 characters of the SHA256 hash of the current directory.
func getPathHash() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}

	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return "unknown"
	}

	hash := sha256.Sum256([]byte(absPath))
	return hex.EncodeToString(hash[:])[:12]
}

// slugifyURL converts a git remote URL into a clean, filesystem-safe identifier.
// It removes protocol prefixes (https://, git@), strips .git suffixes,
// and replaces special characters with hyphens.
//
// Examples:
//   - "https://github.com/user/repo.git" -> "github-com-user-repo"
//   - "git@github.com:user/repo.git" -> "github-com-user-repo"
func slugifyURL(url string) string {
	// Remove common protocol prefixes
	url = removeProtocol(url)

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Replace colons and slashes with hyphens (SSH style: git@host:path)
	url = strings.ReplaceAll(url, ":", "-")
	url = strings.ReplaceAll(url, "/", "-")

	// Replace any remaining special characters with hyphens
	re := regexp.MustCompile(`[^a-zA-Z0-9-]`)
	url = re.ReplaceAllString(url, "-")

	// Collapse multiple hyphens into one
	re = regexp.MustCompile(`-+`)
	url = re.ReplaceAllString(url, "-")

	// Trim leading/trailing hyphens
	url = strings.Trim(url, "-")

	return strings.ToLower(url)
}

// removeProtocol strips protocol prefixes from URLs.
func removeProtocol(url string) string {
	prefixes := []string{
		"https://",
		"http://",
		"git://",
		"ssh://",
		"git@",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(url, prefix) {
			return url[len(prefix):]
		}
	}

	return url
}
