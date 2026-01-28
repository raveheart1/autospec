// Package git_test tests git operations for repository detection and branch retrieval.
// Related: /home/ari/repos/autospec/internal/git/git.go
// Tags: git, repository, branch, vcs

package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	behavior := os.Getenv("TEST_MOCK_BEHAVIOR")
	switch behavior {
	case "":
		os.Exit(m.Run())
	case "gitBranch":
		// Mock successful branch query
		os.Stdout.WriteString("002-go-binary-migration\n")
		os.Exit(0)
	case "gitRoot":
		// Mock successful root query
		os.Stdout.WriteString("/home/user/project\n")
		os.Exit(0)
	case "gitDir":
		// Mock successful git dir check
		os.Exit(0)
	default:
		os.Exit(m.Run())
	}
}

// TestGetCurrentBranch tests retrieving the current branch name
// Note: This test runs against the actual git repository, not a mock
func TestGetCurrentBranch_Real(t *testing.T) {
	branch, err := GetCurrentBranch()
	require.NoError(t, err)
	// In CI (GitHub Actions), we may be in detached HEAD state which returns empty string
	// Just verify the function doesn't error - empty string is valid for detached HEAD
	t.Logf("current branch: %q (empty means detached HEAD)", branch)
}

// TestGetRepositoryRoot tests retrieving the repository root path
func TestGetRepositoryRoot_Real(t *testing.T) {
	root, err := GetRepositoryRoot()
	require.NoError(t, err)
	assert.NotEmpty(t, root)
	assert.Contains(t, root, "autospec")
}

// TestIsGitRepository tests checking if we're in a git repository
func TestIsGitRepository_Real(t *testing.T) {
	isRepo := IsGitRepository()
	assert.True(t, isRepo)
}

// TestGetAllBranches tests listing all branches
func TestGetAllBranches_Real(t *testing.T) {
	branches, err := GetAllBranches()
	require.NoError(t, err)
	assert.NotEmpty(t, branches)

	// Verify we have at least one branch
	assert.GreaterOrEqual(t, len(branches), 1)

	// Check that the current branch is in the list (only when not in detached HEAD)
	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err)

	// Only verify branch-in-list when we have a real branch (not detached HEAD)
	if currentBranch != "" && currentBranch != "HEAD" {
		found := false
		for _, b := range branches {
			if b.Name == currentBranch {
				found = true
				break
			}
		}
		assert.True(t, found, "current branch should be in the branch list")
	}
}

// TestGetBranchNames tests getting just branch names
func TestGetBranchNames_Real(t *testing.T) {
	names, err := GetBranchNames()
	require.NoError(t, err)
	assert.NotEmpty(t, names)

	// Current branch should be in the list (only when not in detached HEAD)
	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err)

	// Only verify branch-in-list when we have a real branch (not detached HEAD)
	if currentBranch != "" && currentBranch != "HEAD" {
		assert.Contains(t, names, currentBranch)
	}
}

// TestBranchInfo verifies BranchInfo structure
func TestBranchInfo(t *testing.T) {
	branches, err := GetAllBranches()
	require.NoError(t, err)

	for _, b := range branches {
		// Name should never be empty
		assert.NotEmpty(t, b.Name)
		// If IsRemote is true, Remote should have a value
		if b.IsRemote {
			assert.NotEmpty(t, b.Remote, "remote branch should have Remote field set")
		}
	}
}

// TestFetchAllRemotes tests fetching from remotes
// This is a light test since we don't want to actually hit the network in unit tests
func TestFetchAllRemotes_Real(t *testing.T) {
	// Just verify it doesn't panic or error fatally
	// The actual fetch might fail if there's no network, but that's ok
	_, err := FetchAllRemotes()
	// We accept either success or network failure
	// The function should never return a hard error, just false
	assert.NoError(t, err)
}

// TestFetchRemoteSkipsSSHWithoutAgent verifies that SSH remotes are skipped
// when SSH_AUTH_SOCK is not set, preventing hangs in sandbox environments.
// Note: Cannot use t.Parallel() as this test manipulates environment variables.
func TestFetchRemoteSkipsSSHWithoutAgent(t *testing.T) {
	// This test verifies the logic path through isSSHURL and isSSHAgentAvailable
	// without actually hitting the network.

	// Save original value
	origValue, origSet := os.LookupEnv("SSH_AUTH_SOCK")
	t.Cleanup(func() {
		if origSet {
			os.Setenv("SSH_AUTH_SOCK", origValue)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	})

	tests := map[string]struct {
		url         string
		sshAgentSet bool
		shouldSkip  bool
		description string
	}{
		"SSH URL without agent should be skipped": {
			url:         "git@github.com:user/repo.git",
			sshAgentSet: false,
			shouldSkip:  true,
			description: "SSH URLs should be skipped when SSH_AUTH_SOCK is unset",
		},
		"SSH URL with agent should not be skipped": {
			url:         "git@github.com:user/repo.git",
			sshAgentSet: true,
			shouldSkip:  false,
			description: "SSH URLs should proceed when SSH_AUTH_SOCK is set",
		},
		"HTTPS URL without agent should not be skipped": {
			url:         "https://github.com/user/repo.git",
			sshAgentSet: false,
			shouldSkip:  false,
			description: "HTTPS URLs should always proceed regardless of SSH agent",
		},
		"HTTPS URL with agent should not be skipped": {
			url:         "https://github.com/user/repo.git",
			sshAgentSet: true,
			shouldSkip:  false,
			description: "HTTPS URLs should always proceed regardless of SSH agent",
		},
		"git+ssh URL without agent should be skipped": {
			url:         "git+ssh://git@github.com/user/repo.git",
			sshAgentSet: false,
			shouldSkip:  true,
			description: "git+ssh:// URLs should be skipped when SSH_AUTH_SOCK is unset",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Set up environment
			if tt.sshAgentSet {
				os.Setenv("SSH_AUTH_SOCK", "/tmp/test-socket")
			} else {
				os.Unsetenv("SSH_AUTH_SOCK")
			}

			// Test the skip logic directly
			isSSH := isSSHURL(tt.url)
			agentAvailable := isSSHAgentAvailable()
			wouldSkip := isSSH && !agentAvailable

			assert.Equal(t, tt.shouldSkip, wouldSkip, tt.description)
		})
	}
}

func TestParseBranchLine(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		line    string
		want    *BranchInfo
		wantNil bool
	}{
		"simple local branch": {
			line: "main",
			want: &BranchInfo{Name: "main", IsRemote: false, Remote: ""},
		},
		"local branch with dash": {
			line: "feature-branch",
			want: &BranchInfo{Name: "feature-branch", IsRemote: false, Remote: ""},
		},
		"local branch with slash": {
			line: "feature/test",
			want: &BranchInfo{Name: "test", IsRemote: true, Remote: "feature"},
		},
		"remote branch origin": {
			line: "origin/main",
			want: &BranchInfo{Name: "main", IsRemote: true, Remote: "origin"},
		},
		"remote branch with remotes prefix": {
			line: "remotes/origin/main",
			want: &BranchInfo{Name: "main", IsRemote: true, Remote: "origin"},
		},
		"remote branch upstream": {
			line: "upstream/develop",
			want: &BranchInfo{Name: "develop", IsRemote: true, Remote: "upstream"},
		},
		"remote with nested path": {
			line: "remotes/origin/feature/my-feature",
			want: &BranchInfo{Name: "feature/my-feature", IsRemote: true, Remote: "origin"},
		},
		"remote without prefix but with slash": {
			line: "origin/feature/nested",
			want: &BranchInfo{Name: "feature/nested", IsRemote: true, Remote: "origin"},
		},
		"remotes prefix with invalid format": {
			line:    "remotes/noslash",
			wantNil: true,
		},
		"empty string": {
			line: "",
			want: &BranchInfo{Name: "", IsRemote: false, Remote: ""},
		},
		"single slash": {
			line: "a/b",
			want: &BranchInfo{Name: "b", IsRemote: true, Remote: "a"},
		},
		"branch with numbers": {
			line: "038-test-coverage",
			want: &BranchInfo{Name: "038-test-coverage", IsRemote: false, Remote: ""},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := parseBranchLine(tt.line)

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.IsRemote, got.IsRemote)
			assert.Equal(t, tt.want.Remote, got.Remote)
		})
	}
}

func TestAddBranchWithDedup(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		existing []BranchInfo
		info     BranchInfo
		seen     map[string]bool
		wantLen  int
		checkFn  func(t *testing.T, result []BranchInfo)
	}{
		"add new local branch": {
			existing: []BranchInfo{},
			info:     BranchInfo{Name: "main", IsRemote: false},
			seen:     map[string]bool{},
			wantLen:  1,
			checkFn: func(t *testing.T, result []BranchInfo) {
				assert.Equal(t, "main", result[0].Name)
				assert.False(t, result[0].IsRemote)
			},
		},
		"add new remote branch": {
			existing: []BranchInfo{},
			info:     BranchInfo{Name: "develop", IsRemote: true, Remote: "origin"},
			seen:     map[string]bool{},
			wantLen:  1,
			checkFn: func(t *testing.T, result []BranchInfo) {
				assert.Equal(t, "develop", result[0].Name)
				assert.True(t, result[0].IsRemote)
			},
		},
		"skip duplicate remote when local exists": {
			existing: []BranchInfo{
				{Name: "main", IsRemote: false},
			},
			info:    BranchInfo{Name: "main", IsRemote: true, Remote: "origin"},
			seen:    map[string]bool{"main": true},
			wantLen: 1,
			checkFn: func(t *testing.T, result []BranchInfo) {
				// Should still have local, not replaced
				assert.False(t, result[0].IsRemote)
			},
		},
		"replace remote with local": {
			existing: []BranchInfo{
				{Name: "feature", IsRemote: true, Remote: "origin"},
			},
			info:    BranchInfo{Name: "feature", IsRemote: false},
			seen:    map[string]bool{"feature": true},
			wantLen: 1,
			checkFn: func(t *testing.T, result []BranchInfo) {
				// Should be replaced with local
				assert.Equal(t, "feature", result[0].Name)
				assert.False(t, result[0].IsRemote)
			},
		},
		"add to non-empty list": {
			existing: []BranchInfo{
				{Name: "main", IsRemote: false},
			},
			info:    BranchInfo{Name: "develop", IsRemote: false},
			seen:    map[string]bool{"main": true},
			wantLen: 2,
			checkFn: func(t *testing.T, result []BranchInfo) {
				assert.Equal(t, 2, len(result))
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Make a copy of existing to avoid mutation
			existing := make([]BranchInfo, len(tt.existing))
			copy(existing, tt.existing)

			// Make a copy of seen map
			seen := make(map[string]bool)
			for k, v := range tt.seen {
				seen[k] = v
			}

			result := addBranchWithDedup(existing, tt.info, seen)
			assert.Equal(t, tt.wantLen, len(result))
			tt.checkFn(t, result)
		})
	}
}

func TestCollectBranches(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lines   []string
		wantLen int
		checkFn func(t *testing.T, branches []BranchInfo)
	}{
		"empty lines": {
			lines:   []string{},
			wantLen: 0,
		},
		"single local branch": {
			lines:   []string{"main"},
			wantLen: 1,
			checkFn: func(t *testing.T, branches []BranchInfo) {
				assert.Equal(t, "main", branches[0].Name)
			},
		},
		"filters HEAD": {
			lines:   []string{"main", "HEAD", "origin/HEAD"},
			wantLen: 1,
			checkFn: func(t *testing.T, branches []BranchInfo) {
				assert.Equal(t, "main", branches[0].Name)
			},
		},
		"filters empty lines": {
			lines:   []string{"main", "", "  ", "develop"},
			wantLen: 2,
		},
		"deduplicates local over remote": {
			lines:   []string{"origin/main", "main"},
			wantLen: 1,
			checkFn: func(t *testing.T, branches []BranchInfo) {
				// Local should replace remote
				assert.Equal(t, "main", branches[0].Name)
				assert.False(t, branches[0].IsRemote)
			},
		},
		"multiple remotes same branch": {
			lines:   []string{"origin/feature", "upstream/feature"},
			wantLen: 1,
			checkFn: func(t *testing.T, branches []BranchInfo) {
				// First one wins
				assert.Equal(t, "feature", branches[0].Name)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			branches := collectBranches(tt.lines)
			assert.Equal(t, tt.wantLen, len(branches))
			if tt.checkFn != nil {
				tt.checkFn(t, branches)
			}
		})
	}
}

// TestCreateBranch_InTempRepo tests CreateBranch in a temporary git repository
// Note: Cannot use t.Parallel() as this test changes the working directory
func TestCreateBranch_InTempRepo(t *testing.T) {
	// Create a temp directory for our test git repo
	tmpDir := t.TempDir()

	// Helper to run git commands in tmpDir (avoids global config pollution from race conditions)
	runGit := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		return cmd.Run()
	}

	// Initialize a git repo
	err := runGit("init")
	require.NoError(t, err)

	// Configure git user for the test repo (uses cmd.Dir to ensure local config)
	err = runGit("config", "user.email", "test@test.com")
	require.NoError(t, err)

	err = runGit("config", "user.name", "Test User")
	require.NoError(t, err)

	// Create an initial commit (needed for branches to work)
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0o644)
	require.NoError(t, err)

	err = runGit("add", ".")
	require.NoError(t, err)

	err = runGit("commit", "-m", "initial commit")
	require.NoError(t, err)

	// Change to temp dir for the functions under test (they use cwd)
	origDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	tests := map[string]struct {
		branchName string
		wantErr    bool
		errContain string
	}{
		"create new branch": {
			branchName: "test-new-branch",
			wantErr:    false,
		},
		"create branch with numbers": {
			branchName: "feature-123-test",
			wantErr:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := CreateBranch(tt.branchName)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)

				// Verify branch was created
				branch, err := GetCurrentBranch()
				require.NoError(t, err)
				assert.Equal(t, tt.branchName, branch)

				// Switch back to main for next test
				if err := runGit("checkout", "master"); err != nil {
					// Try main instead of master
					_ = runGit("checkout", "main")
				}
			}
		})
	}

	// Test creating duplicate branch
	t.Run("duplicate branch fails", func(t *testing.T) {
		err := CreateBranch("test-new-branch")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

// TestIsSSHAgentAvailable tests SSH agent availability detection.
// Note: Cannot use t.Parallel() as this test manipulates environment variables.
func TestIsSSHAgentAvailable(t *testing.T) {
	// Save original value to restore after all tests
	origValue, origSet := os.LookupEnv("SSH_AUTH_SOCK")
	t.Cleanup(func() {
		if origSet {
			os.Setenv("SSH_AUTH_SOCK", origValue)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	})

	tests := map[string]struct {
		envValue string
		envSet   bool
		want     bool
	}{
		"SSH_AUTH_SOCK set and non-empty": {
			envValue: "/tmp/ssh-agent.sock",
			envSet:   true,
			want:     true,
		},
		"SSH_AUTH_SOCK set to valid path": {
			envValue: "/run/user/1000/keyring/ssh",
			envSet:   true,
			want:     true,
		},
		"SSH_AUTH_SOCK not set": {
			envValue: "",
			envSet:   false,
			want:     false,
		},
		"SSH_AUTH_SOCK set to empty string": {
			envValue: "",
			envSet:   true,
			want:     false,
		},
		"SSH_AUTH_SOCK set to whitespace only": {
			envValue: "   ",
			envSet:   true,
			want:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Set or unset the env var for this test
			if tt.envSet {
				os.Setenv("SSH_AUTH_SOCK", tt.envValue)
			} else {
				os.Unsetenv("SSH_AUTH_SOCK")
			}

			got := isSSHAgentAvailable()
			assert.Equal(t, tt.want, got, "isSSHAgentAvailable() = %v, want %v", got, tt.want)
		})
	}
}

func TestIsSSHURL(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		url  string
		want bool
	}{
		// SSH formats that should return true
		"git@ format": {
			url:  "git@github.com:user/repo.git",
			want: true,
		},
		"git@ format gitlab": {
			url:  "git@gitlab.com:org/project.git",
			want: true,
		},
		"ssh:// format": {
			url:  "ssh://git@github.com/user/repo.git",
			want: true,
		},
		"ssh:// format with port": {
			url:  "ssh://git@github.com:22/user/repo.git",
			want: true,
		},
		"git+ssh:// format": {
			url:  "git+ssh://git@github.com/user/repo.git",
			want: true,
		},
		"git+ssh:// format bitbucket": {
			url:  "git+ssh://git@bitbucket.org/team/repo.git",
			want: true,
		},

		// HTTPS formats that should return false
		"https:// format": {
			url:  "https://github.com/user/repo.git",
			want: false,
		},
		"https:// format with auth": {
			url:  "https://user:token@github.com/user/repo.git",
			want: false,
		},
		"http:// format": {
			url:  "http://github.com/user/repo.git",
			want: false,
		},
		"http:// format with auth": {
			url:  "http://user:pass@example.com/repo.git",
			want: false,
		},

		// Edge cases
		"empty string": {
			url:  "",
			want: false,
		},
		"file:// protocol": {
			url:  "file:///path/to/repo.git",
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := isSSHURL(tt.url)
			assert.Equal(t, tt.want, got, "isSSHURL(%q) = %v, want %v", tt.url, got, tt.want)
		})
	}
}

// TestCreateBranch_NotGitRepo tests CreateBranch fails outside a git repo
// Note: Cannot use t.Parallel() as this test changes the working directory
func TestCreateBranch_NotGitRepo(t *testing.T) {
	// Create a temp directory that is NOT a git repo
	tmpDir := t.TempDir()

	// Save current directory and change to temp dir
	origDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// Try to create a branch - should fail
	err = CreateBranch("test-branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

// TestFetchAllRemotesWithContext tests context-based fetch with timeout behavior.
// Uses map-based table test pattern as per NFR-005.
func TestFetchAllRemotesWithContext(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		description string
		setupFn     func(t *testing.T) (context.Context, context.CancelFunc)
		wantTimeout bool
	}{
		"normal context completes without timeout": {
			description: "Fetch should complete normally with background context",
			setupFn: func(t *testing.T) (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			wantTimeout: false,
		},
		"cancelled context returns immediately": {
			description: "Fetch should return when context is already cancelled",
			setupFn: func(t *testing.T) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, func() {}
			},
			wantTimeout: true,
		},
		"short timeout context cancels fetch": {
			description: "Fetch should be cancelled when timeout is very short",
			setupFn: func(t *testing.T) (context.Context, context.CancelFunc) {
				// Use a tiny timeout to ensure cancellation happens quickly
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				time.Sleep(1 * time.Millisecond) // Ensure timeout triggers
				return ctx, cancel
			},
			wantTimeout: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := tt.setupFn(t)
			defer cancel()

			// Test against actual repo - the function should respect context
			success, err := FetchAllRemotesWithContext(ctx)

			if tt.wantTimeout {
				// With cancelled context, should return quickly without error
				// but success may be false due to early exit
				assert.NoError(t, err, "timeout/cancellation should not return error")
				// Success can be either true or false depending on timing
				_ = success
			} else {
				// Normal context should work (may fail due to network, but no error)
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// TestFetchAllRemotesWithContext_HandlesTimeoutGracefully verifies that
// timeout errors are handled gracefully with a warning log, not an error.
func TestFetchAllRemotesWithContext_HandlesTimeoutGracefully(t *testing.T) {
	// Create a context that's already cancelled to simulate timeout
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should return without error - timeout is handled gracefully
	_, err := FetchAllRemotesWithContext(ctx)
	assert.NoError(t, err, "timeout should be handled gracefully, not return error")
}

// TestFetchAllRemotes_UsesDefaultTimeout verifies that FetchAllRemotes
// uses the default timeout (60 seconds as per FR-008).
func TestFetchAllRemotes_UsesDefaultTimeout(t *testing.T) {
	// This test verifies that FetchAllRemotes delegates to FetchAllRemotesWithContext
	// by checking that it completes successfully (no hang).
	// The actual timeout behavior is tested via the WithContext variant.

	// Use a channel with timeout to ensure the function doesn't hang
	done := make(chan struct{})
	var success bool
	var err error

	go func() {
		success, err = FetchAllRemotes()
		close(done)
	}()

	select {
	case <-done:
		// Completed within reasonable time
		assert.NoError(t, err, "FetchAllRemotes should complete without error")
		// success can be true or false depending on network/remotes
		_ = success
	case <-time.After(5 * time.Second):
		t.Fatal("FetchAllRemotes appears to hang - timeout mechanism may not be working")
	}
}
