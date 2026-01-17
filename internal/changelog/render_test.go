package changelog

import (
	"strings"
	"testing"
)

func TestRenderMarkdownString(t *testing.T) {
	tests := map[string]struct {
		changelog   *Changelog
		contains    []string
		notContains []string
	}{
		"single version with all categories": {
			changelog: &Changelog{
				Project: "testproject",
				Versions: []Version{
					{
						Version: "1.0.0",
						Date:    "2026-01-15",
						Changes: Changes{
							Added:      []string{"Feature A"},
							Changed:    []string{"Feature B"},
							Deprecated: []string{"Feature C"},
							Removed:    []string{"Feature D"},
							Fixed:      []string{"Bug E"},
							Security:   []string{"Vuln F"},
						},
					},
				},
			},
			contains: []string{
				"# Changelog",
				"All notable changes to testproject",
				"## [1.0.0] - 2026-01-15",
				"### Added",
				"- Feature A",
				"### Changed",
				"- Feature B",
				"### Deprecated",
				"- Feature C",
				"### Removed",
				"- Feature D",
				"### Fixed",
				"- Bug E",
				"### Security",
				"- Vuln F",
			},
			notContains: []string{},
		},
		"unreleased version": {
			changelog: &Changelog{
				Project: "testproject",
				Versions: []Version{
					{
						Version: "unreleased",
						Changes: Changes{
							Added: []string{"New feature"},
						},
					},
				},
			},
			contains: []string{
				"## [Unreleased]",
				"### Added",
				"- New feature",
			},
			notContains: []string{
				" - ", // No date for unreleased
			},
		},
		"empty categories omitted": {
			changelog: &Changelog{
				Project: "testproject",
				Versions: []Version{
					{
						Version: "1.0.0",
						Date:    "2026-01-15",
						Changes: Changes{
							Added: []string{"Feature A"},
							Fixed: []string{"Bug B"},
						},
					},
				},
			},
			contains: []string{
				"### Added",
				"### Fixed",
			},
			notContains: []string{
				"### Changed",
				"### Deprecated",
				"### Removed",
				"### Security",
			},
		},
		"multiple entries per category": {
			changelog: &Changelog{
				Project: "testproject",
				Versions: []Version{
					{
						Version: "1.0.0",
						Date:    "2026-01-15",
						Changes: Changes{
							Added: []string{"Feature A", "Feature B", "Feature C"},
						},
					},
				},
			},
			contains: []string{
				"- Feature A",
				"- Feature B",
				"- Feature C",
			},
			notContains: []string{},
		},
		"multiple versions": {
			changelog: &Changelog{
				Project: "testproject",
				Versions: []Version{
					{
						Version: "unreleased",
						Changes: Changes{
							Added: []string{"Upcoming feature"},
						},
					},
					{
						Version: "1.1.0",
						Date:    "2026-01-16",
						Changes: Changes{
							Added: []string{"Feature 1.1"},
						},
					},
					{
						Version: "1.0.0",
						Date:    "2026-01-15",
						Changes: Changes{
							Added: []string{"Initial release"},
						},
					},
				},
			},
			contains: []string{
				"## [Unreleased]",
				"## [1.1.0] - 2026-01-16",
				"## [1.0.0] - 2026-01-15",
			},
			notContains: []string{},
		},
		"entries with markdown": {
			changelog: &Changelog{
				Project: "testproject",
				Versions: []Version{
					{
						Version: "1.0.0",
						Date:    "2026-01-15",
						Changes: Changes{
							Added: []string{
								"New `command` with **bold** text",
								"Support for [links](https://example.com)",
							},
						},
					},
				},
			},
			contains: []string{
				"- New `command` with **bold** text",
				"- Support for [links](https://example.com)",
			},
			notContains: []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := RenderMarkdownString(tt.changelog)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, result)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("expected output NOT to contain %q, got:\n%s", notExpected, result)
				}
			}
		})
	}
}

func TestRenderMarkdownIdempotent(t *testing.T) {
	changelog := &Changelog{
		Project: "testproject",
		Versions: []Version{
			{
				Version: "unreleased",
				Changes: Changes{
					Added: []string{"New feature"},
				},
			},
			{
				Version: "1.0.0",
				Date:    "2026-01-15",
				Changes: Changes{
					Added:   []string{"Feature A"},
					Fixed:   []string{"Bug B"},
					Changed: []string{"Change C"},
				},
			},
		},
	}

	result1, err := RenderMarkdownString(changelog)
	if err != nil {
		t.Fatalf("first render failed: %v", err)
	}

	result2, err := RenderMarkdownString(changelog)
	if err != nil {
		t.Fatalf("second render failed: %v", err)
	}

	if result1 != result2 {
		t.Errorf("idempotency check failed:\nFirst:\n%s\nSecond:\n%s", result1, result2)
	}
}

func TestRenderMarkdownCategoryOrder(t *testing.T) {
	changelog := &Changelog{
		Project: "testproject",
		Versions: []Version{
			{
				Version: "1.0.0",
				Date:    "2026-01-15",
				Changes: Changes{
					Security:   []string{"S"},
					Fixed:      []string{"F"},
					Removed:    []string{"R"},
					Deprecated: []string{"D"},
					Changed:    []string{"C"},
					Added:      []string{"A"},
				},
			},
		},
	}

	result, err := RenderMarkdownString(changelog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Category order should be: Added, Changed, Deprecated, Removed, Fixed, Security
	addedPos := strings.Index(result, "### Added")
	changedPos := strings.Index(result, "### Changed")
	deprecatedPos := strings.Index(result, "### Deprecated")
	removedPos := strings.Index(result, "### Removed")
	fixedPos := strings.Index(result, "### Fixed")
	securityPos := strings.Index(result, "### Security")

	if addedPos == -1 || changedPos == -1 || deprecatedPos == -1 ||
		removedPos == -1 || fixedPos == -1 || securityPos == -1 {
		t.Fatal("not all categories found in output")
	}

	if !(addedPos < changedPos && changedPos < deprecatedPos &&
		deprecatedPos < removedPos && removedPos < fixedPos &&
		fixedPos < securityPos) {
		t.Error("categories are not in the correct order")
	}
}

func TestRenderMarkdownFooterLinks(t *testing.T) {
	tests := map[string]struct {
		changelog   *Changelog
		contains    []string
		notContains []string
	}{
		"autospec project with multiple versions": {
			changelog: &Changelog{
				Project: "autospec",
				Versions: []Version{
					{
						Version: "unreleased",
						Changes: Changes{Added: []string{"A"}},
					},
					{
						Version: "1.1.0",
						Date:    "2026-01-16",
						Changes: Changes{Added: []string{"B"}},
					},
					{
						Version: "1.0.0",
						Date:    "2026-01-15",
						Changes: Changes{Added: []string{"C"}},
					},
				},
			},
			contains: []string{
				"[Unreleased]: https://github.com/ariel-frischer/autospec/compare/v1.1.0...HEAD",
				"[1.1.0]: https://github.com/ariel-frischer/autospec/compare/v1.0.0...v1.1.0",
				"[1.0.0]: https://github.com/ariel-frischer/autospec/releases/tag/v1.0.0",
			},
			notContains: []string{},
		},
		"unknown project has no comparison links": {
			changelog: &Changelog{
				Project: "unknownproject",
				Versions: []Version{
					{
						Version: "1.0.0",
						Date:    "2026-01-15",
						Changes: Changes{Added: []string{"A"}},
					},
				},
			},
			contains:    []string{},
			notContains: []string{"[1.0.0]:"},
		},
		"single unreleased version": {
			changelog: &Changelog{
				Project: "autospec",
				Versions: []Version{
					{
						Version: "unreleased",
						Changes: Changes{Added: []string{"A"}},
					},
				},
			},
			contains:    []string{},
			notContains: []string{"[Unreleased]:"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := RenderMarkdownString(tt.changelog)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected output to contain %q", expected)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("expected output NOT to contain %q", notExpected)
				}
			}
		})
	}
}

func TestRenderMarkdownEmptyChangelog(t *testing.T) {
	changelog := &Changelog{
		Project:  "testproject",
		Versions: []Version{},
	}

	result, err := RenderMarkdownString(changelog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "# Changelog") {
		t.Error("expected header in output")
	}
}
