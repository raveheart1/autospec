package prereqs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeContext(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T) (specsDir string, cleanup func())
		opts        Options
		wantErr     bool
		errContains string
		validate    func(t *testing.T, ctx *Context)
	}{
		"feature dir present with all files": {
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "prereqs-test-*")
				require.NoError(t, err)

				featureDir := filepath.Join(tmpDir, "001-test-feature")
				require.NoError(t, os.MkdirAll(featureDir, 0o755))

				// Create all required files
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "spec.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "plan.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "tasks.yaml"), []byte("test"), 0o644))

				// Set env var for feature detection
				oldEnv := os.Getenv("SPECIFY_FEATURE")
				os.Setenv("SPECIFY_FEATURE", "001-test-feature")

				return tmpDir, func() {
					os.Setenv("SPECIFY_FEATURE", oldEnv)
					os.RemoveAll(tmpDir)
				}
			},
			opts: Options{
				RequireSpec:  true,
				RequirePlan:  true,
				RequireTasks: true,
				IncludeTasks: true,
			},
			wantErr: false,
			validate: func(t *testing.T, ctx *Context) {
				assert.NotEmpty(t, ctx.FeatureDir)
				assert.True(t, strings.HasSuffix(ctx.FeatureDir, "001-test-feature"))
				assert.True(t, strings.HasSuffix(ctx.FeatureSpec, "spec.yaml"))
				assert.True(t, strings.HasSuffix(ctx.ImplPlan, "plan.yaml"))
				assert.True(t, strings.HasSuffix(ctx.TasksFile, "tasks.yaml"))
				assert.Contains(t, ctx.AutospecVersion, "autospec")
				assert.NotEmpty(t, ctx.CreatedDate)
				assert.Contains(t, ctx.AvailableDocs, "spec.yaml")
				assert.Contains(t, ctx.AvailableDocs, "plan.yaml")
				assert.Contains(t, ctx.AvailableDocs, "tasks.yaml")
			},
		},
		"missing spec.yaml when required": {
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "prereqs-test-*")
				require.NoError(t, err)

				featureDir := filepath.Join(tmpDir, "001-test-feature")
				require.NoError(t, os.MkdirAll(featureDir, 0o755))

				// Only create plan.yaml, not spec.yaml
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "plan.yaml"), []byte("test"), 0o644))

				oldEnv := os.Getenv("SPECIFY_FEATURE")
				os.Setenv("SPECIFY_FEATURE", "001-test-feature")

				return tmpDir, func() {
					os.Setenv("SPECIFY_FEATURE", oldEnv)
					os.RemoveAll(tmpDir)
				}
			},
			opts: Options{
				RequireSpec: true,
			},
			wantErr:     true,
			errContains: "no spec.yaml found",
		},
		"missing plan.yaml when required": {
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "prereqs-test-*")
				require.NoError(t, err)

				featureDir := filepath.Join(tmpDir, "001-test-feature")
				require.NoError(t, os.MkdirAll(featureDir, 0o755))

				// Only create spec.yaml, not plan.yaml
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "spec.yaml"), []byte("test"), 0o644))

				oldEnv := os.Getenv("SPECIFY_FEATURE")
				os.Setenv("SPECIFY_FEATURE", "001-test-feature")

				return tmpDir, func() {
					os.Setenv("SPECIFY_FEATURE", oldEnv)
					os.RemoveAll(tmpDir)
				}
			},
			opts: Options{
				RequirePlan: true,
			},
			wantErr:     true,
			errContains: "no plan.yaml found",
		},
		"missing tasks.yaml when required": {
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "prereqs-test-*")
				require.NoError(t, err)

				featureDir := filepath.Join(tmpDir, "001-test-feature")
				require.NoError(t, os.MkdirAll(featureDir, 0o755))

				// Create spec and plan but not tasks
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "spec.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "plan.yaml"), []byte("test"), 0o644))

				oldEnv := os.Getenv("SPECIFY_FEATURE")
				os.Setenv("SPECIFY_FEATURE", "001-test-feature")

				return tmpDir, func() {
					os.Setenv("SPECIFY_FEATURE", oldEnv)
					os.RemoveAll(tmpDir)
				}
			},
			opts: Options{
				RequireTasks: true,
			},
			wantErr:     true,
			errContains: "no tasks.yaml found",
		},
		"paths only mode skips validation": {
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "prereqs-test-*")
				require.NoError(t, err)

				featureDir := filepath.Join(tmpDir, "001-test-feature")
				require.NoError(t, os.MkdirAll(featureDir, 0o755))

				oldEnv := os.Getenv("SPECIFY_FEATURE")
				os.Setenv("SPECIFY_FEATURE", "001-test-feature")

				return tmpDir, func() {
					os.Setenv("SPECIFY_FEATURE", oldEnv)
					os.RemoveAll(tmpDir)
				}
			},
			opts: Options{
				PathsOnly:    true,
				RequireSpec:  true,
				RequirePlan:  true,
				RequireTasks: true,
			},
			wantErr: false,
			validate: func(t *testing.T, ctx *Context) {
				// Should have paths even though files don't exist
				assert.NotEmpty(t, ctx.FeatureDir)
				assert.Contains(t, ctx.AutospecVersion, "autospec")
				assert.NotEmpty(t, ctx.CreatedDate)
			},
		},
		"checklists directory detected": {
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "prereqs-test-*")
				require.NoError(t, err)

				featureDir := filepath.Join(tmpDir, "001-test-feature")
				checklistsDir := filepath.Join(featureDir, "checklists")
				require.NoError(t, os.MkdirAll(checklistsDir, 0o755))

				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "spec.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "plan.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(checklistsDir, "test.yaml"), []byte("test"), 0o644))

				oldEnv := os.Getenv("SPECIFY_FEATURE")
				os.Setenv("SPECIFY_FEATURE", "001-test-feature")

				return tmpDir, func() {
					os.Setenv("SPECIFY_FEATURE", oldEnv)
					os.RemoveAll(tmpDir)
				}
			},
			opts: Options{
				RequirePlan: true,
			},
			wantErr: false,
			validate: func(t *testing.T, ctx *Context) {
				assert.Contains(t, ctx.AvailableDocs, "checklists/")
			},
		},
		"empty checklists directory not included": {
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "prereqs-test-*")
				require.NoError(t, err)

				featureDir := filepath.Join(tmpDir, "001-test-feature")
				checklistsDir := filepath.Join(featureDir, "checklists")
				require.NoError(t, os.MkdirAll(checklistsDir, 0o755))

				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "spec.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "plan.yaml"), []byte("test"), 0o644))

				oldEnv := os.Getenv("SPECIFY_FEATURE")
				os.Setenv("SPECIFY_FEATURE", "001-test-feature")

				return tmpDir, func() {
					os.Setenv("SPECIFY_FEATURE", oldEnv)
					os.RemoveAll(tmpDir)
				}
			},
			opts: Options{
				RequirePlan: true,
			},
			wantErr: false,
			validate: func(t *testing.T, ctx *Context) {
				assert.NotContains(t, ctx.AvailableDocs, "checklists/")
			},
		},
		"default plan required when no require flags": {
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "prereqs-test-*")
				require.NoError(t, err)

				featureDir := filepath.Join(tmpDir, "001-test-feature")
				require.NoError(t, os.MkdirAll(featureDir, 0o755))

				// Create only spec.yaml - plan.yaml should be required by default
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "spec.yaml"), []byte("test"), 0o644))

				oldEnv := os.Getenv("SPECIFY_FEATURE")
				os.Setenv("SPECIFY_FEATURE", "001-test-feature")

				return tmpDir, func() {
					os.Setenv("SPECIFY_FEATURE", oldEnv)
					os.RemoveAll(tmpDir)
				}
			},
			opts: Options{
				// No require flags - should default to requiring plan
			},
			wantErr:     true,
			errContains: "no plan.yaml found",
		},
		"include tasks only when flag set": {
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "prereqs-test-*")
				require.NoError(t, err)

				featureDir := filepath.Join(tmpDir, "001-test-feature")
				require.NoError(t, os.MkdirAll(featureDir, 0o755))

				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "spec.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "plan.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(featureDir, "tasks.yaml"), []byte("test"), 0o644))

				oldEnv := os.Getenv("SPECIFY_FEATURE")
				os.Setenv("SPECIFY_FEATURE", "001-test-feature")

				return tmpDir, func() {
					os.Setenv("SPECIFY_FEATURE", oldEnv)
					os.RemoveAll(tmpDir)
				}
			},
			opts: Options{
				RequirePlan:  true,
				IncludeTasks: false,
			},
			wantErr: false,
			validate: func(t *testing.T, ctx *Context) {
				// tasks.yaml should not be in available docs without IncludeTasks
				assert.NotContains(t, ctx.AvailableDocs, "tasks.yaml")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			specsDir, cleanup := tt.setup(t)
			defer cleanup()

			opts := tt.opts
			opts.SpecsDir = specsDir

			ctx, err := ComputeContext(opts)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, ctx)

			if tt.validate != nil {
				tt.validate(t, ctx)
			}
		})
	}
}

func TestContextFields(t *testing.T) {
	tests := map[string]struct {
		ctx      *Context
		validate func(t *testing.T, ctx *Context)
	}{
		"all fields populated": {
			ctx: &Context{
				FeatureDir:      "/path/to/specs/001-test",
				FeatureSpec:     "/path/to/specs/001-test/spec.yaml",
				ImplPlan:        "/path/to/specs/001-test/plan.yaml",
				TasksFile:       "/path/to/specs/001-test/tasks.yaml",
				AutospecVersion: "autospec 0.9.0",
				CreatedDate:     "2024-01-15T10:30:00Z",
				IsGitRepo:       true,
				AvailableDocs:   []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
			},
			validate: func(t *testing.T, ctx *Context) {
				assert.Equal(t, "/path/to/specs/001-test", ctx.FeatureDir)
				assert.Equal(t, "/path/to/specs/001-test/spec.yaml", ctx.FeatureSpec)
				assert.Equal(t, "/path/to/specs/001-test/plan.yaml", ctx.ImplPlan)
				assert.Equal(t, "/path/to/specs/001-test/tasks.yaml", ctx.TasksFile)
				assert.Equal(t, "autospec 0.9.0", ctx.AutospecVersion)
				assert.Equal(t, "2024-01-15T10:30:00Z", ctx.CreatedDate)
				assert.True(t, ctx.IsGitRepo)
				assert.Len(t, ctx.AvailableDocs, 3)
			},
		},
		"empty context": {
			ctx: &Context{},
			validate: func(t *testing.T, ctx *Context) {
				assert.Empty(t, ctx.FeatureDir)
				assert.Empty(t, ctx.FeatureSpec)
				assert.Empty(t, ctx.ImplPlan)
				assert.Empty(t, ctx.TasksFile)
				assert.Empty(t, ctx.AutospecVersion)
				assert.Empty(t, ctx.CreatedDate)
				assert.False(t, ctx.IsGitRepo)
				assert.Empty(t, ctx.AvailableDocs)
			},
		},
		"partial context": {
			ctx: &Context{
				FeatureDir:      "/path/to/specs/001-test",
				AutospecVersion: "autospec 0.9.0",
				IsGitRepo:       false,
			},
			validate: func(t *testing.T, ctx *Context) {
				assert.NotEmpty(t, ctx.FeatureDir)
				assert.Empty(t, ctx.FeatureSpec)
				assert.NotEmpty(t, ctx.AutospecVersion)
				assert.False(t, ctx.IsGitRepo)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.validate(t, tt.ctx)
		})
	}
}

func TestOptions(t *testing.T) {
	tests := map[string]struct {
		opts     Options
		validate func(t *testing.T, opts Options)
	}{
		"all flags set": {
			opts: Options{
				SpecsDir:     "./custom-specs",
				RequireSpec:  true,
				RequirePlan:  true,
				RequireTasks: true,
				IncludeTasks: true,
				PathsOnly:    true,
			},
			validate: func(t *testing.T, opts Options) {
				assert.Equal(t, "./custom-specs", opts.SpecsDir)
				assert.True(t, opts.RequireSpec)
				assert.True(t, opts.RequirePlan)
				assert.True(t, opts.RequireTasks)
				assert.True(t, opts.IncludeTasks)
				assert.True(t, opts.PathsOnly)
			},
		},
		"default options": {
			opts: Options{},
			validate: func(t *testing.T, opts Options) {
				assert.Empty(t, opts.SpecsDir)
				assert.False(t, opts.RequireSpec)
				assert.False(t, opts.RequirePlan)
				assert.False(t, opts.RequireTasks)
				assert.False(t, opts.IncludeTasks)
				assert.False(t, opts.PathsOnly)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.validate(t, tt.opts)
		})
	}
}

func TestBuildContextFromMetadata(t *testing.T) {
	t.Run("builds context from nil metadata", func(t *testing.T) {
		ctx := buildContextFromMetadata(nil, true)

		assert.Empty(t, ctx.FeatureDir)
		assert.Empty(t, ctx.FeatureSpec)
		assert.Empty(t, ctx.ImplPlan)
		assert.Empty(t, ctx.TasksFile)
		assert.Contains(t, ctx.AutospecVersion, "autospec")
		assert.NotEmpty(t, ctx.CreatedDate)
		assert.True(t, ctx.IsGitRepo)
		assert.Empty(t, ctx.AvailableDocs)
	})

	t.Run("preserves isGitRepo flag", func(t *testing.T) {
		ctx := buildContextFromMetadata(nil, false)
		assert.False(t, ctx.IsGitRepo)

		ctx = buildContextFromMetadata(nil, true)
		assert.True(t, ctx.IsGitRepo)
	})
}

func TestValidateContextPaths(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T) (*Context, func())
		opts        Options
		wantErr     bool
		errContains string
	}{
		"empty feature dir": {
			setup: func(t *testing.T) (*Context, func()) {
				return &Context{}, func() {}
			},
			opts:        Options{},
			wantErr:     true,
			errContains: "feature directory not detected",
		},
		"non-existent feature dir": {
			setup: func(t *testing.T) (*Context, func()) {
				return &Context{
					FeatureDir: "/non/existent/path",
				}, func() {}
			},
			opts:        Options{},
			wantErr:     true,
			errContains: "feature directory not found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cleanup := tt.setup(t)
			defer cleanup()

			err := validateContextPaths(ctx, tt.opts)

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

func TestComputeAvailableDocs(t *testing.T) {
	tests := map[string]struct {
		setup        func(t *testing.T) (*Context, func())
		includeTasks bool
		wantDocs     []string
	}{
		"all docs present with include tasks": {
			setup: func(t *testing.T) (*Context, func()) {
				tmpDir, err := os.MkdirTemp("", "docs-test-*")
				require.NoError(t, err)

				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "spec.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "plan.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "tasks.yaml"), []byte("test"), 0o644))

				ctx := &Context{
					FeatureDir:  tmpDir,
					FeatureSpec: filepath.Join(tmpDir, "spec.yaml"),
					ImplPlan:    filepath.Join(tmpDir, "plan.yaml"),
					TasksFile:   filepath.Join(tmpDir, "tasks.yaml"),
				}

				return ctx, func() { os.RemoveAll(tmpDir) }
			},
			includeTasks: true,
			wantDocs:     []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
		},
		"tasks excluded when not requested": {
			setup: func(t *testing.T) (*Context, func()) {
				tmpDir, err := os.MkdirTemp("", "docs-test-*")
				require.NoError(t, err)

				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "spec.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "plan.yaml"), []byte("test"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "tasks.yaml"), []byte("test"), 0o644))

				ctx := &Context{
					FeatureDir:  tmpDir,
					FeatureSpec: filepath.Join(tmpDir, "spec.yaml"),
					ImplPlan:    filepath.Join(tmpDir, "plan.yaml"),
					TasksFile:   filepath.Join(tmpDir, "tasks.yaml"),
				}

				return ctx, func() { os.RemoveAll(tmpDir) }
			},
			includeTasks: false,
			wantDocs:     []string{"spec.yaml", "plan.yaml"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cleanup := tt.setup(t)
			defer cleanup()

			docs := computeAvailableDocs(ctx, tt.includeTasks)

			for _, wantDoc := range tt.wantDocs {
				assert.Contains(t, docs, wantDoc)
			}
		})
	}
}
