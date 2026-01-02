package git

import (
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createInMemoryRepoWithCommit creates an in-memory git repository with an initial commit
func createInMemoryRepoWithCommit(t *testing.T) (*git.Repository, string) {
	t.Helper()

	store := memory.NewStorage()
	repo, err := git.Init(store, nil)
	require.NoError(t, err)

	// Create a commit
	worktree, err := repo.Worktree()
	if err != nil {
		// For in-memory repos without filesystem, we need to create commit differently
		// Use low-level API to create a commit object
		commitHash := createCommitInMemory(t, repo, store)
		return repo, commitHash.String()
	}

	_ = worktree // silence unused variable if we get here
	return repo, ""
}

// createCommitInMemory creates a commit using go-git's low-level API for in-memory repos
func createCommitInMemory(t *testing.T, repo *git.Repository, store *memory.Storage) plumbing.Hash {
	t.Helper()

	// Create a blob
	blob := store.NewEncodedObject()
	blob.SetType(plumbing.BlobObject)
	writer, err := blob.Writer()
	require.NoError(t, err)
	_, err = writer.Write([]byte("test content"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)
	blobHash, err := store.SetEncodedObject(blob)
	require.NoError(t, err)

	// Create a tree with the blob
	tree := object.Tree{
		Entries: []object.TreeEntry{
			{
				Name: "test.txt",
				Mode: 0o100644,
				Hash: blobHash,
			},
		},
	}
	treeObj := store.NewEncodedObject()
	err = tree.Encode(treeObj)
	require.NoError(t, err)
	treeHash, err := store.SetEncodedObject(treeObj)
	require.NoError(t, err)

	// Create a commit
	commit := &object.Commit{
		Author: object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
		Message:  "Initial commit",
		TreeHash: treeHash,
	}
	commitObj := store.NewEncodedObject()
	err = commit.Encode(commitObj)
	require.NoError(t, err)
	commitHash, err := store.SetEncodedObject(commitObj)
	require.NoError(t, err)

	// Update HEAD to point to the commit via a branch
	mainRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), commitHash)
	err = store.SetReference(mainRef)
	require.NoError(t, err)

	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	err = store.SetReference(headRef)
	require.NoError(t, err)

	return commitHash
}

// createSecondCommit creates a second commit on top of the first
func createSecondCommit(t *testing.T, store *memory.Storage, parentHash plumbing.Hash) plumbing.Hash {
	t.Helper()

	// Create a new blob
	blob := store.NewEncodedObject()
	blob.SetType(plumbing.BlobObject)
	writer, err := blob.Writer()
	require.NoError(t, err)
	_, err = writer.Write([]byte("updated content"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)
	blobHash, err := store.SetEncodedObject(blob)
	require.NoError(t, err)

	// Create a tree with the blob
	tree := object.Tree{
		Entries: []object.TreeEntry{
			{
				Name: "test.txt",
				Mode: 0o100644,
				Hash: blobHash,
			},
		},
	}
	treeObj := store.NewEncodedObject()
	err = tree.Encode(treeObj)
	require.NoError(t, err)
	treeHash, err := store.SetEncodedObject(treeObj)
	require.NoError(t, err)

	// Create a commit with parent
	commit := &object.Commit{
		Author: object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
		Message:      "Second commit",
		TreeHash:     treeHash,
		ParentHashes: []plumbing.Hash{parentHash},
	}
	commitObj := store.NewEncodedObject()
	err = commit.Encode(commitObj)
	require.NoError(t, err)
	commitHash, err := store.SetEncodedObject(commitObj)
	require.NoError(t, err)

	// Update main branch to point to new commit
	mainRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), commitHash)
	err = store.SetReference(mainRef)
	require.NoError(t, err)

	return commitHash
}

func TestCaptureGitState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupRepo func(t *testing.T) (*memory.Storage, *git.Repository)
		wantErr   bool
		check     func(t *testing.T, state *GitState)
	}{
		"captures branch and commit SHA": {
			setupRepo: func(t *testing.T) (*memory.Storage, *git.Repository) {
				store := memory.NewStorage()
				repo, err := git.Init(store, nil)
				require.NoError(t, err)
				createCommitInMemory(t, repo, store)
				return store, repo
			},
			wantErr: false,
			check: func(t *testing.T, state *GitState) {
				assert.Equal(t, "main", state.BranchName)
				assert.Len(t, state.CommitSHA, 40) // SHA is 40 hex chars
				assert.False(t, state.CapturedAt.IsZero())
			},
		},
		"detects detached HEAD state": {
			setupRepo: func(t *testing.T) (*memory.Storage, *git.Repository) {
				store := memory.NewStorage()
				repo, err := git.Init(store, nil)
				require.NoError(t, err)
				commitHash := createCommitInMemory(t, repo, store)

				// Set HEAD to point directly to commit (detached)
				headRef := plumbing.NewHashReference(plumbing.HEAD, commitHash)
				err = store.SetReference(headRef)
				require.NoError(t, err)

				return store, repo
			},
			wantErr: false,
			check: func(t *testing.T, state *GitState) {
				assert.Equal(t, "detached", state.BranchName)
				assert.Len(t, state.CommitSHA, 40)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store, repo := tt.setupRepo(t)
			opener := NewInMemoryOpener(store, repo)

			state, err := CaptureGitState(opener, "/fake/path")

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, state)
			tt.check(t, state)
		})
	}
}

func TestCaptureGitState_OpenerError(t *testing.T) {
	t.Parallel()

	// Use InMemoryOpener without a repo configured
	opener := &InMemoryOpener{}

	state, err := CaptureGitState(opener, "/nonexistent/path")

	assert.Error(t, err)
	assert.Nil(t, state)
	assert.Contains(t, err.Error(), "opening git repository")
}

func TestCompareGitStates(t *testing.T) {
	t.Parallel()

	baseTime := time.Now().UTC()

	tests := map[string]struct {
		initial      *GitState
		final        *GitState
		wantWarnings []StateWarning
	}{
		"no warnings when commit created on same branch": {
			initial: &GitState{
				CommitSHA:  "abc123abc123abc123abc123abc123abc123abc1",
				BranchName: "main",
				CapturedAt: baseTime,
			},
			final: &GitState{
				CommitSHA:  "def456def456def456def456def456def456def4",
				BranchName: "main",
				CapturedAt: baseTime.Add(time.Minute),
			},
			wantWarnings: nil,
		},
		"warning when no new commit created": {
			initial: &GitState{
				CommitSHA:  "abc123abc123abc123abc123abc123abc123abc1",
				BranchName: "main",
				CapturedAt: baseTime,
			},
			final: &GitState{
				CommitSHA:  "abc123abc123abc123abc123abc123abc123abc1",
				BranchName: "main",
				CapturedAt: baseTime.Add(time.Minute),
			},
			wantWarnings: []StateWarning{
				{Level: "warning", Message: "no new commit was created during workflow"},
			},
		},
		"serious warning when branch changed": {
			initial: &GitState{
				CommitSHA:  "abc123abc123abc123abc123abc123abc123abc1",
				BranchName: "main",
				CapturedAt: baseTime,
			},
			final: &GitState{
				CommitSHA:  "def456def456def456def456def456def456def4",
				BranchName: "feature",
				CapturedAt: baseTime.Add(time.Minute),
			},
			wantWarnings: []StateWarning{
				{Level: "serious", Message: "branch changed during workflow: main -> feature"},
			},
		},
		"nil initial state returns no warnings": {
			initial:      nil,
			final:        &GitState{CommitSHA: "abc123", BranchName: "main"},
			wantWarnings: nil,
		},
		"nil final state returns no warnings": {
			initial:      &GitState{CommitSHA: "abc123", BranchName: "main"},
			final:        nil,
			wantWarnings: nil,
		},
		"both nil returns no warnings": {
			initial:      nil,
			final:        nil,
			wantWarnings: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			warnings := CompareGitStates(tt.initial, tt.final)

			if tt.wantWarnings == nil {
				assert.Empty(t, warnings)
				return
			}

			assert.Equal(t, len(tt.wantWarnings), len(warnings))
			for i, w := range tt.wantWarnings {
				assert.Equal(t, w.Level, warnings[i].Level)
				assert.Equal(t, w.Message, warnings[i].Message)
			}
		})
	}
}

func TestGitState_Fields(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	state := GitState{
		CommitSHA:  "1234567890abcdef1234567890abcdef12345678",
		BranchName: "feature-branch",
		CapturedAt: now,
	}

	assert.Equal(t, "1234567890abcdef1234567890abcdef12345678", state.CommitSHA)
	assert.Equal(t, "feature-branch", state.BranchName)
	assert.Equal(t, now, state.CapturedAt)
}

func TestStateWarning_Fields(t *testing.T) {
	t.Parallel()

	warning := StateWarning{
		Level:   "serious",
		Message: "test message",
	}

	assert.Equal(t, "serious", warning.Level)
	assert.Equal(t, "test message", warning.Message)
}

func TestDefaultOpener_Interface(t *testing.T) {
	t.Parallel()

	// Verify DefaultOpener implements Opener interface
	var _ Opener = &DefaultOpener{}
}

func TestInMemoryOpener_Interface(t *testing.T) {
	t.Parallel()

	// Verify InMemoryOpener implements Opener interface
	var _ Opener = &InMemoryOpener{}
}

func TestNewInMemoryOpener(t *testing.T) {
	t.Parallel()

	store := memory.NewStorage()
	repo, err := git.Init(store, nil)
	require.NoError(t, err)

	opener := NewInMemoryOpener(store, repo)

	assert.NotNil(t, opener)
	assert.Equal(t, store, opener.storage)
	assert.Equal(t, repo, opener.repo)
}
