package commands

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/prereqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTemplate(t *testing.T) {
	tests := map[string]struct {
		content     string
		ctx         *prereqs.Context
		wantErr     bool
		errContains string
		validate    func(t *testing.T, output []byte)
	}{
		"successful rendering with all variables": {
			content: "Feature: {{.FeatureDir}}\nSpec: {{.FeatureSpec}}\nPlan: {{.ImplPlan}}",
			ctx: &prereqs.Context{
				FeatureDir:  "/path/to/specs/001-test",
				FeatureSpec: "/path/to/specs/001-test/spec.yaml",
				ImplPlan:    "/path/to/specs/001-test/plan.yaml",
			},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Contains(t, string(output), "/path/to/specs/001-test")
				assert.Contains(t, string(output), "spec.yaml")
				assert.Contains(t, string(output), "plan.yaml")
			},
		},
		"rendering with partial context": {
			content: "Version: {{.AutospecVersion}}\nDate: {{.CreatedDate}}",
			ctx: &prereqs.Context{
				AutospecVersion: "autospec 0.9.0",
				CreatedDate:     "2024-01-15T10:30:00Z",
			},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Contains(t, string(output), "autospec 0.9.0")
				assert.Contains(t, string(output), "2024-01-15T10:30:00Z")
			},
		},
		"rendering with IsGitRepo boolean": {
			content: "Git repo: {{.IsGitRepo}}",
			ctx: &prereqs.Context{
				IsGitRepo: true,
			},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Contains(t, string(output), "true")
			},
		},
		"rendering with AvailableDocs list": {
			content: "Docs: {{range .AvailableDocs}}{{.}} {{end}}",
			ctx: &prereqs.Context{
				AvailableDocs: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
			},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Contains(t, string(output), "spec.yaml")
				assert.Contains(t, string(output), "plan.yaml")
				assert.Contains(t, string(output), "tasks.yaml")
			},
		},
		"nil context returns error": {
			content:     "Test {{.FeatureDir}}",
			ctx:         nil,
			wantErr:     true,
			errContains: "prereqs context is nil",
		},
		"empty template renders successfully": {
			content: "",
			ctx:     &prereqs.Context{},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Empty(t, output)
			},
		},
		"template with no variables renders as-is": {
			content: "This is plain text with no variables",
			ctx:     &prereqs.Context{},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Equal(t, "This is plain text with no variables", string(output))
			},
		},
		"invalid template syntax returns error": {
			content:     "Invalid {{.Unclosed",
			ctx:         &prereqs.Context{},
			wantErr:     true,
			errContains: "parsing template",
		},
		"missing field reference renders empty string": {
			content: "Dir: {{.FeatureDir}}",
			ctx:     &prereqs.Context{},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Equal(t, "Dir: ", string(output))
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			output, err := RenderTemplate([]byte(tt.content), tt.ctx)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, output)
			}
		})
	}
}

func TestGetRequiredVars(t *testing.T) {
	tests := map[string]struct {
		commandName string
		wantVars    []string
	}{
		"plan command requires spec context": {
			commandName: "autospec.plan",
			wantVars:    []string{"FeatureDir", "FeatureSpec", "AutospecVersion", "CreatedDate"},
		},
		"tasks command requires plan context": {
			commandName: "autospec.tasks",
			wantVars:    []string{"FeatureDir", "FeatureSpec", "ImplPlan", "AutospecVersion", "CreatedDate"},
		},
		"implement command requires tasks context": {
			commandName: "autospec.implement",
			wantVars:    []string{"FeatureDir", "TasksFile"},
		},
		"constitution command requires minimal context": {
			commandName: "autospec.constitution",
			wantVars:    []string{"AutospecVersion", "CreatedDate"},
		},
		"checklist command requires spec context": {
			commandName: "autospec.checklist",
			wantVars:    []string{"FeatureDir", "FeatureSpec"},
		},
		"clarify command requires spec context": {
			commandName: "autospec.clarify",
			wantVars:    []string{"FeatureDir", "FeatureSpec"},
		},
		"analyze command requires spec context": {
			commandName: "autospec.analyze",
			wantVars:    []string{"FeatureDir", "FeatureSpec"},
		},
		"specify command requires no prereqs": {
			commandName: "autospec.specify",
			wantVars:    []string{},
		},
		"unknown command returns empty slice": {
			commandName: "unknown.command",
			wantVars:    []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetRequiredVars(tt.commandName)
			assert.Equal(t, tt.wantVars, got)
		})
	}
}

func TestValidateRequirements(t *testing.T) {
	tests := map[string]struct {
		commandName string
		ctx         *prereqs.Context
		wantErr     bool
		errContains string
	}{
		"valid plan command context": {
			commandName: "autospec.plan",
			ctx: &prereqs.Context{
				FeatureDir:      "/path/to/specs/001-test",
				FeatureSpec:     "/path/to/specs/001-test/spec.yaml",
				AutospecVersion: "autospec 0.9.0",
				CreatedDate:     "2024-01-15T10:30:00Z",
			},
			wantErr: false,
		},
		"missing required field for plan command": {
			commandName: "autospec.plan",
			ctx: &prereqs.Context{
				FeatureDir: "/path/to/specs/001-test",
				// Missing FeatureSpec, AutospecVersion, CreatedDate
			},
			wantErr:     true,
			errContains: "missing required context for autospec.plan",
		},
		"valid implement command context": {
			commandName: "autospec.implement",
			ctx: &prereqs.Context{
				FeatureDir: "/path/to/specs/001-test",
				TasksFile:  "/path/to/specs/001-test/tasks.yaml",
			},
			wantErr: false,
		},
		"missing tasks file for implement command": {
			commandName: "autospec.implement",
			ctx: &prereqs.Context{
				FeatureDir: "/path/to/specs/001-test",
				// Missing TasksFile
			},
			wantErr:     true,
			errContains: "TasksFile",
		},
		"valid constitution command context": {
			commandName: "autospec.constitution",
			ctx: &prereqs.Context{
				AutospecVersion: "autospec 0.9.0",
				CreatedDate:     "2024-01-15T10:30:00Z",
			},
			wantErr: false,
		},
		"unknown command accepts any context": {
			commandName: "unknown.command",
			ctx:         &prereqs.Context{},
			wantErr:     false,
		},
		"nil context returns error": {
			commandName: "autospec.plan",
			ctx:         nil,
			wantErr:     true,
			errContains: "prereqs context is nil",
		},
		"specify command accepts empty context": {
			commandName: "autospec.specify",
			ctx:         &prereqs.Context{},
			wantErr:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateRequirements(tt.commandName, tt.ctx)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestHasContextField(t *testing.T) {
	tests := map[string]struct {
		ctx   *prereqs.Context
		field string
		want  bool
	}{
		"FeatureDir populated": {
			ctx:   &prereqs.Context{FeatureDir: "/path"},
			field: "FeatureDir",
			want:  true,
		},
		"FeatureDir empty": {
			ctx:   &prereqs.Context{},
			field: "FeatureDir",
			want:  false,
		},
		"FeatureSpec populated": {
			ctx:   &prereqs.Context{FeatureSpec: "/path/spec.yaml"},
			field: "FeatureSpec",
			want:  true,
		},
		"ImplPlan populated": {
			ctx:   &prereqs.Context{ImplPlan: "/path/plan.yaml"},
			field: "ImplPlan",
			want:  true,
		},
		"TasksFile populated": {
			ctx:   &prereqs.Context{TasksFile: "/path/tasks.yaml"},
			field: "TasksFile",
			want:  true,
		},
		"AutospecVersion populated": {
			ctx:   &prereqs.Context{AutospecVersion: "autospec 0.9.0"},
			field: "AutospecVersion",
			want:  true,
		},
		"CreatedDate populated": {
			ctx:   &prereqs.Context{CreatedDate: "2024-01-15T10:30:00Z"},
			field: "CreatedDate",
			want:  true,
		},
		"IsGitRepo always returns true": {
			ctx:   &prereqs.Context{},
			field: "IsGitRepo",
			want:  true,
		},
		"AvailableDocs always returns true": {
			ctx:   &prereqs.Context{},
			field: "AvailableDocs",
			want:  true,
		},
		"unknown field returns false": {
			ctx:   &prereqs.Context{},
			field: "UnknownField",
			want:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := hasContextField(tt.ctx, tt.field)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRenderAndValidate(t *testing.T) {
	tests := map[string]struct {
		commandName string
		content     string
		ctx         *prereqs.Context
		wantErr     bool
		errContains string
		validate    func(t *testing.T, output []byte)
	}{
		"successful render and validate for plan command": {
			commandName: "autospec.plan",
			content:     "Feature: {{.FeatureDir}}\nSpec: {{.FeatureSpec}}",
			ctx: &prereqs.Context{
				FeatureDir:      "/path/to/specs/001-test",
				FeatureSpec:     "/path/to/specs/001-test/spec.yaml",
				AutospecVersion: "autospec 0.9.0",
				CreatedDate:     "2024-01-15T10:30:00Z",
			},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Contains(t, string(output), "/path/to/specs/001-test")
			},
		},
		"validation fails before rendering": {
			commandName: "autospec.plan",
			content:     "Feature: {{.FeatureDir}}",
			ctx:         &prereqs.Context{
				// Missing required fields
			},
			wantErr:     true,
			errContains: "missing required context",
		},
		"specify command renders with empty context": {
			commandName: "autospec.specify",
			content:     "Static content with {{.AutospecVersion}}",
			ctx: &prereqs.Context{
				AutospecVersion: "autospec 0.9.0",
			},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Contains(t, string(output), "autospec 0.9.0")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			output, err := RenderAndValidate(tt.commandName, []byte(tt.content), tt.ctx)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, output)
			}
		})
	}
}

func TestTemplateEdgeCases(t *testing.T) {
	tests := map[string]struct {
		content  string
		ctx      *prereqs.Context
		wantErr  bool
		validate func(t *testing.T, output []byte)
	}{
		"template with markdown code blocks": {
			content: "```bash\nautospec prereqs --json\n```\nPath: {{.FeatureDir}}",
			ctx: &prereqs.Context{
				FeatureDir: "/specs/001-test",
			},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				result := string(output)
				assert.Contains(t, result, "```bash")
				assert.Contains(t, result, "/specs/001-test")
			},
		},
		"template with yaml blocks": {
			content: "```yaml\nplan:\n  branch: \"{{.FeatureDir}}\"\n```",
			ctx: &prereqs.Context{
				FeatureDir: "/specs/001-test",
			},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Contains(t, string(output), "/specs/001-test")
			},
		},
		"template with conditional": {
			content: "{{if .IsGitRepo}}In git repo{{else}}Not in git repo{{end}}",
			ctx: &prereqs.Context{
				IsGitRepo: true,
			},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Equal(t, "In git repo", string(output))
			},
		},
		"template with special characters in path": {
			content: "Path: {{.FeatureDir}}",
			ctx: &prereqs.Context{
				FeatureDir: "/path/with spaces/and-special_chars",
			},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				assert.Contains(t, string(output), "/path/with spaces/and-special_chars")
			},
		},
		"template preserves newlines and formatting": {
			content: "Line 1\n\nLine 3\n  Indented",
			ctx:     &prereqs.Context{},
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				lines := strings.Split(string(output), "\n")
				assert.Len(t, lines, 4)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			output, err := RenderTemplate([]byte(tt.content), tt.ctx)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, output)
			}
		})
	}
}
