package dag

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConflictMarkers(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"no conflicts": {
			input:    "line 1\nline 2\nline 3\n",
			expected: "",
		},
		"single conflict block": {
			input: `before
<<<<<<< HEAD
line from head
=======
line from branch
>>>>>>> feature-branch
after
`,
			expected: `Line 2:
<<<<<<< HEAD
line from head
=======
line from branch
>>>>>>> feature-branch
`,
		},
		"multiple conflict blocks": {
			input: `<<<<<<< HEAD
first conflict
=======
first resolution
>>>>>>> branch
some code
<<<<<<< HEAD
second conflict
=======
second resolution
>>>>>>> branch
`,
			expected: `Line 1:
<<<<<<< HEAD
first conflict
=======
first resolution
>>>>>>> branch

---

Line 7:
<<<<<<< HEAD
second conflict
=======
second resolution
>>>>>>> branch
`,
		},
		"conflict with content before and after": {
			input: `package main

func main() {
<<<<<<< HEAD
    fmt.Println("hello")
=======
    fmt.Println("world")
>>>>>>> feature
}
`,
			expected: `Line 4:
<<<<<<< HEAD
    fmt.Println("hello")
=======
    fmt.Println("world")
>>>>>>> feature
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := parseConflictMarkers(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expected {
				t.Errorf("result mismatch:\ngot:\n%s\nwant:\n%s", result, tc.expected)
			}
		})
	}
}

func TestHasConflictMarkers(t *testing.T) {
	tests := map[string]struct {
		content  string
		expected bool
	}{
		"no markers": {
			content:  "normal code\nno conflicts here\n",
			expected: false,
		},
		"has start marker": {
			content:  "<<<<<<< HEAD\n",
			expected: true,
		},
		"has separator": {
			content:  "=======\n",
			expected: true,
		},
		"has end marker": {
			content:  ">>>>>>> branch\n",
			expected: true,
		},
		"full conflict": {
			content:  "<<<<<<< HEAD\nours\n=======\ntheirs\n>>>>>>> branch\n",
			expected: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := hasConflictMarkers(tc.content)
			if result != tc.expected {
				t.Errorf("got %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestLookupSpecInfo(t *testing.T) {
	dag := &DAGConfig{
		Layers: []Layer{
			{
				ID: "L0",
				Features: []Feature{
					{ID: "spec-a", Description: "Feature A description"},
					{ID: "spec-b", Description: ""},
				},
			},
			{
				ID: "L1",
				Features: []Feature{
					{ID: "spec-c", Description: "Feature C description"},
				},
			},
		},
	}

	tests := map[string]struct {
		dag      *DAGConfig
		specID   string
		wantName string
		wantDesc string
	}{
		"found with description": {
			dag:      dag,
			specID:   "spec-a",
			wantName: "spec-a",
			wantDesc: "Feature A description",
		},
		"found without description": {
			dag:      dag,
			specID:   "spec-b",
			wantName: "spec-b",
			wantDesc: "",
		},
		"found in second layer": {
			dag:      dag,
			specID:   "spec-c",
			wantName: "spec-c",
			wantDesc: "Feature C description",
		},
		"not found": {
			dag:      dag,
			specID:   "spec-unknown",
			wantName: "spec-unknown",
			wantDesc: "",
		},
		"nil dag": {
			dag:      nil,
			specID:   "spec-a",
			wantName: "spec-a",
			wantDesc: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotName, gotDesc := lookupSpecInfo(tc.dag, tc.specID)
			if gotName != tc.wantName {
				t.Errorf("name mismatch: got %q, want %q", gotName, tc.wantName)
			}
			if gotDesc != tc.wantDesc {
				t.Errorf("description mismatch: got %q, want %q", gotDesc, tc.wantDesc)
			}
		})
	}
}

func TestBuildConflictContext(t *testing.T) {
	tests := map[string]struct {
		fileContent  string
		specID       string
		dag          *DAGConfig
		sourceBranch string
		targetBranch string
		wantErr      bool
		checkResult  func(t *testing.T, ctx *ConflictContext)
	}{
		"builds context with all fields": {
			fileContent: `<<<<<<< HEAD
ours
=======
theirs
>>>>>>> feature
`,
			specID: "spec-a",
			dag: &DAGConfig{
				Layers: []Layer{{
					ID: "L0",
					Features: []Feature{
						{ID: "spec-a", Description: "Test feature"},
					},
				}},
			},
			sourceBranch: "feature-branch",
			targetBranch: "main",
			wantErr:      false,
			checkResult: func(t *testing.T, ctx *ConflictContext) {
				if ctx.SpecID != "spec-a" {
					t.Errorf("SpecID mismatch: got %s", ctx.SpecID)
				}
				if ctx.SpecDescription != "Test feature" {
					t.Errorf("SpecDescription mismatch: got %s", ctx.SpecDescription)
				}
				if ctx.SourceBranch != "feature-branch" {
					t.Errorf("SourceBranch mismatch: got %s", ctx.SourceBranch)
				}
				if ctx.TargetBranch != "main" {
					t.Errorf("TargetBranch mismatch: got %s", ctx.TargetBranch)
				}
				if !strings.Contains(ctx.ConflictDiff, "<<<<<<<") {
					t.Error("ConflictDiff should contain conflict markers")
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "conflict.go")
			if err := os.WriteFile(filePath, []byte(tc.fileContent), 0644); err != nil {
				t.Fatalf("writing test file: %v", err)
			}

			resolver := NewConflictResolver(tmpDir, nil, nil)
			ctx, err := resolver.BuildConflictContext(
				"conflict.go", tc.specID, tc.dag, tc.sourceBranch, tc.targetBranch,
			)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tc.checkResult(t, ctx)
		})
	}
}

func TestBuildAgentPrompt(t *testing.T) {
	conflict := &ConflictContext{
		FilePath:        "pkg/service.go",
		SpecID:          "spec-123",
		SpecDescription: "Add authentication to API",
		SourceBranch:    "feature/auth",
		TargetBranch:    "main",
		ConflictDiff: `<<<<<<< HEAD
func Auth() {}
=======
func Authenticate() {}
>>>>>>> feature/auth
`,
	}

	prompt := buildAgentPrompt(conflict)

	// Check essential elements are present
	checks := []string{
		"spec-123",
		"Add authentication to API",
		"feature/auth",
		"main",
		"pkg/service.go",
		"<<<<<<<",
		"git add",
		"conflict",
	}

	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("prompt should contain %q, got:\n%s", check, prompt)
		}
	}
}

func TestOutputManualContext(t *testing.T) {
	var buf bytes.Buffer
	resolver := NewConflictResolver("/repo", nil, &buf)

	conflicts := []*ConflictContext{
		{
			FilePath:        "file1.go",
			SpecID:          "spec-a",
			SpecDescription: "Feature A",
			SourceBranch:    "feature-a",
			TargetBranch:    "main",
			ConflictDiff:    "<<<<<<< HEAD\nours\n=======\ntheirs\n>>>>>>> branch\n",
		},
		{
			FilePath:        "file2.go",
			SpecID:          "spec-a",
			SpecDescription: "Feature A",
			SourceBranch:    "feature-a",
			TargetBranch:    "main",
			ConflictDiff:    "<<<<<<< HEAD\nours2\n=======\ntheirs2\n>>>>>>> branch\n",
		},
	}

	resolver.OutputManualContext(conflicts)
	output := buf.String()

	// Check for essential output elements
	checks := []string{
		"MERGE CONFLICT",
		"Manual Resolution Required",
		"file1.go",
		"file2.go",
		"spec-a",
		"Feature A",
		"feature-a",
		"main",
		"<<<<<<<",
		"git add",
		"autospec dag merge --continue",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output should contain %q", check)
		}
	}
}

func TestBuildAllConflictContexts(t *testing.T) {
	tests := map[string]struct {
		files          map[string]string
		conflictedList []string
		specID         string
		wantCount      int
		wantErr        bool
	}{
		"single file": {
			files: map[string]string{
				"a.go": "<<<<<<< HEAD\nours\n=======\ntheirs\n>>>>>>> branch\n",
			},
			conflictedList: []string{"a.go"},
			specID:         "spec-1",
			wantCount:      1,
		},
		"multiple files": {
			files: map[string]string{
				"a.go": "<<<<<<< HEAD\na\n=======\nb\n>>>>>>> branch\n",
				"b.go": "<<<<<<< HEAD\nc\n=======\nd\n>>>>>>> branch\n",
			},
			conflictedList: []string{"a.go", "b.go"},
			specID:         "spec-1",
			wantCount:      2,
		},
		"missing file returns error": {
			files:          map[string]string{},
			conflictedList: []string{"nonexistent.go"},
			specID:         "spec-1",
			wantErr:        true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for filename, content := range tc.files {
				path := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("writing file: %v", err)
				}
			}

			resolver := NewConflictResolver(tmpDir, nil, nil)
			contexts, err := resolver.BuildAllConflictContexts(
				tc.conflictedList, tc.specID, nil, "source", "target",
			)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(contexts) != tc.wantCount {
				t.Errorf("got %d contexts, want %d", len(contexts), tc.wantCount)
			}
		})
	}
}

func TestVerifyConflictResolved(t *testing.T) {
	tests := map[string]struct {
		content string
		wantErr bool
	}{
		"resolved file": {
			content: "package main\n\nfunc main() {\n\tfmt.Println(\"merged\")\n}\n",
			wantErr: false,
		},
		"still has start marker": {
			content: "<<<<<<< HEAD\ncode\n",
			wantErr: true,
		},
		"still has separator": {
			content: "code\n=======\nmore code\n",
			wantErr: true,
		},
		"still has end marker": {
			content: "code\n>>>>>>> branch\n",
			wantErr: true,
		},
		"full conflict still present": {
			content: "<<<<<<< HEAD\nours\n=======\ntheirs\n>>>>>>> branch\n",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := "test.go"
			fullPath := filepath.Join(tmpDir, filePath)
			if err := os.WriteFile(fullPath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("writing file: %v", err)
			}

			err := verifyConflictResolved(tmpDir, filePath)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNewConflictResolver(t *testing.T) {
	tests := map[string]struct {
		stdout  *bytes.Buffer
		wantNil bool
	}{
		"with stdout": {
			stdout:  &bytes.Buffer{},
			wantNil: false,
		},
		"nil stdout uses default": {
			stdout:  nil,
			wantNil: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			resolver := NewConflictResolver("/repo", nil, tc.stdout)
			if resolver == nil {
				t.Error("resolver should not be nil")
			}
			if resolver.repoRoot != "/repo" {
				t.Errorf("repoRoot mismatch: got %s", resolver.repoRoot)
			}
		})
	}
}

func TestResolveWithAgentNoAgent(t *testing.T) {
	resolver := NewConflictResolver("/repo", nil, nil)

	err := resolver.ResolveWithAgent(context.Background(), []*ConflictContext{
		{FilePath: "test.go"},
	})

	if err == nil {
		t.Error("expected error when no agent configured")
	}

	if !strings.Contains(err.Error(), "no agent configured") {
		t.Errorf("error should mention no agent configured, got: %v", err)
	}
}

func TestDetectConflicts(t *testing.T) {
	// This is a wrapper test - the underlying DetectConflictedFiles
	// is already tested in merge_test.go
	tmpDir := t.TempDir()
	result := DetectConflicts(tmpDir)
	if result != nil {
		t.Errorf("expected nil for non-git directory, got: %v", result)
	}
}
