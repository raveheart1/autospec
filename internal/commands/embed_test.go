// Package commands_test tests embedded template file access and retrieval.
// Related: /home/ari/repos/autospec/internal/commands/embed.go
// Tags: commands, embed, templates, filesystem

package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateFS_Contains_Templates(t *testing.T) {
	entries, err := TemplateFS.ReadDir(".")
	require.NoError(t, err, "should read embedded directory")
	assert.NotEmpty(t, entries, "should contain embedded templates")
}

func TestTemplateFS_ReadFile_Specify(t *testing.T) {
	content, err := TemplateFS.ReadFile("autospec.specify.md")
	require.NoError(t, err, "should read autospec.specify.md")
	assert.NotEmpty(t, content, "template should have content")
	assert.Contains(t, string(content), "description:", "should have frontmatter")
}

func TestTemplateFS_ReadFile_Plan(t *testing.T) {
	content, err := TemplateFS.ReadFile("autospec.plan.md")
	require.NoError(t, err, "should read autospec.plan.md")
	assert.NotEmpty(t, content, "template should have content")
}

func TestTemplateFS_ReadFile_Tasks(t *testing.T) {
	content, err := TemplateFS.ReadFile("autospec.tasks.md")
	require.NoError(t, err, "should read autospec.tasks.md")
	assert.NotEmpty(t, content, "template should have content")
}

func TestTemplateFS_ReadFile_NotFound(t *testing.T) {
	_, err := TemplateFS.ReadFile("nonexistent.md")
	assert.Error(t, err, "should error on non-existent file")
}

func TestGetTemplateNames(t *testing.T) {
	names, err := GetTemplateNames()
	require.NoError(t, err)
	assert.Contains(t, names, "autospec.specify", "should include specify template")
	assert.Contains(t, names, "autospec.plan", "should include plan template")
	assert.Contains(t, names, "autospec.tasks", "should include tasks template")
}

func TestGetTemplateByFilename(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		filename string
		wantErr  bool
		wantLen  int // minimum expected content length, 0 means just check not empty
	}{
		"valid autospec.specify.md": {
			filename: "autospec.specify.md",
			wantErr:  false,
			wantLen:  100, // templates have substantial content
		},
		"valid autospec.plan.md": {
			filename: "autospec.plan.md",
			wantErr:  false,
			wantLen:  100,
		},
		"valid autospec.tasks.md": {
			filename: "autospec.tasks.md",
			wantErr:  false,
			wantLen:  100,
		},
		"valid autospec.implement.md": {
			filename: "autospec.implement.md",
			wantErr:  false,
			wantLen:  100,
		},
		"nonexistent file": {
			filename: "nonexistent.md",
			wantErr:  true,
		},
		"empty filename": {
			filename: "",
			wantErr:  true,
		},
		"invalid extension": {
			filename: "autospec.specify.txt",
			wantErr:  true,
		},
		"path traversal attempt": {
			filename: "../etc/passwd",
			wantErr:  true,
		},
		"directory traversal": {
			filename: "foo/bar.md",
			wantErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			content, err := GetTemplateByFilename(tt.filename)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, content)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, content)
			if tt.wantLen > 0 {
				assert.GreaterOrEqual(t, len(content), tt.wantLen,
					"template content should have at least %d bytes", tt.wantLen)
			}
		})
	}
}

func TestGetTemplate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name    string
		wantErr bool
	}{
		"valid specify template": {
			name:    "autospec.specify",
			wantErr: false,
		},
		"valid plan template": {
			name:    "autospec.plan",
			wantErr: false,
		},
		"nonexistent template": {
			name:    "nonexistent",
			wantErr: true,
		},
		"empty name": {
			name:    "",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			content, err := GetTemplate(tt.name)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, content)
		})
	}
}

func TestIsAutospecTemplate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		filename string
		want     bool
	}{
		"autospec.specify.md": {
			filename: "autospec.specify.md",
			want:     true,
		},
		"autospec.plan.md": {
			filename: "autospec.plan.md",
			want:     true,
		},
		"autospec.tasks.md": {
			filename: "autospec.tasks.md",
			want:     true,
		},
		"autospec.implement.md": {
			filename: "autospec.implement.md",
			want:     true,
		},
		"non-autospec template": {
			filename: "custom-command.md",
			want:     false,
		},
		"empty filename": {
			filename: "",
			want:     false,
		},
		"just autospec prefix": {
			filename: "autospec",
			want:     false,
		},
		"autospec with dot": {
			filename: "autospec.",
			want:     true, // has "autospec." prefix
		},
		"path with autospec filename": {
			filename: "/some/path/autospec.clarify.md",
			want:     true, // filepath.Base extracts the filename
		},
		"path with non-autospec filename": {
			filename: "/some/path/custom.md",
			want:     false,
		},
		"AUTOSPEC uppercase": {
			filename: "AUTOSPEC.specify.md",
			want:     false, // case sensitive
		},
		"mixed case": {
			filename: "Autospec.specify.md",
			want:     false, // case sensitive
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsAutospecTemplate(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}
