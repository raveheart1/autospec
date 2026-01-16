# CHANGELOG.yaml Feature Plan

High-level plan for migrating changelog management to YAML as the single source of truth.

## Overview

Replace manual CHANGELOG.md editing with a structured CHANGELOG.yaml that:
- Serves as the authoritative changelog source
- Generates CHANGELOG.md automatically
- Provides release description content for GitHub CI
- Powers a new CLI command for viewing changelog entries
- Integrates with `update` and `ck` commands for user awareness

## File Locations

| File | Location | Purpose |
|------|----------|---------|
| CHANGELOG.yaml | `.autospec/changelog.yaml` | Source of truth (project-scoped) |
| CHANGELOG.md | `./CHANGELOG.md` (root) | Generated output for users |
| Generator script | `.release/generate-changelog.go` | Build-time generation |

**Rationale**: `.autospec/` already holds project-level config and state. Keeps root clean.

## CHANGELOG.yaml Schema

```yaml
# .autospec/changelog.yaml
project: autospec
versions:
  - version: unreleased
    changes:
      added:
        - "Description of new feature"
      changed:
        - "Description of change"
      fixed:
        - "Bug fix description"
      removed:
        - "Removed item"

  - version: "0.5.0"
    date: "2025-01-15"
    changes:
      added:
        - "DAG workflow orchestration"
        - "Parallel spec execution with --parallel flag"
      changed:
        - "BREAKING: dag run is now idempotent"
      fixed:
        - "Worktree cleanup on failure"
```

**Design notes:**
- Categories match Keep a Changelog standard (added/changed/fixed/removed/deprecated/security)
- `unreleased` is a special version string (no date)
- Entries are user-facing strings (no internal jargon)

## Component Plan

### 1. CHANGELOG.yaml Parser (`internal/changelog/`)

New package to handle YAML changelog operations:

- `internal/changelog/schema.go` - Go structs for YAML schema
- `internal/changelog/parse.go` - Load and validate CHANGELOG.yaml
- `internal/changelog/render.go` - Render to markdown format
- `internal/changelog/query.go` - Query entries by version, category, or count

Key functions:
- `Load(path string) (*Changelog, error)` - Parse YAML file
- `RenderMarkdown(c *Changelog) string` - Generate full CHANGELOG.md content
- `RenderVersion(v *Version) string` - Render single version block
- `GetLatestN(c *Changelog, n int) []Entry` - Get N most recent entries

### 2. CLI Command: `autospec changelog`

New subcommand at `internal/cli/util/changelog.go`:

```bash
# Show recent entries (default: 5)
autospec changelog

# Show specific version
autospec changelog v0.5.0

# Show all unreleased changes
autospec changelog unreleased

# Show last N entries across all versions
autospec changelog --last 10

# Plain output for scripting
autospec changelog --plain
```

**Integration with binary:** Embed CHANGELOG.yaml at build time using `//go:embed` so users can view changelog without network access.

### 3. CHANGELOG.md Generation

**Option A: Build-time (recommended)**
- Add `make changelog` target that runs generator
- `.release/generate-changelog.go` reads YAML, writes MD
- Pre-commit hook ensures MD stays in sync
- CI validates MD matches YAML

**Option B: CI-only**
- Generate MD only during release
- Less dev friction but MD may drift in repo

**Recommendation:** Option A with pre-commit validation.

### 4. GitHub Release Workflow Updates

Modify `.github/workflows/release.yml`:

```yaml
- name: Extract release notes from CHANGELOG.yaml
  run: |
    go run .release/extract-changelog.go --version ${{ github.ref_name }} \
      --input .autospec/changelog.yaml \
      --output .release/notes.md
```

Replace current bash script (`.release/extract-changelog.sh`) with Go-based extraction that reads YAML directly.

**Benefits:**
- Structured parsing (no regex/awk fragility)
- Consistent rendering logic shared with CLI
- Can add YAML validation step

### 5. Integration: `update` Command

After successful update, show what's new:

```
✓ Successfully updated to v0.6.0

What's new in v0.6.0:
  • DAG workflow orchestration
  • Parallel spec execution

Run 'autospec changelog v0.6.0' for full details.
```

**Implementation:** `internal/cli/util/update.go:151` - Add call to changelog query after successful install.

### 6. Integration: `ck` (check) Command

When update is available, show preview:

```
✓ Update available: v0.5.0 → v0.6.0

Highlights in v0.6.0:
  • DAG workflow orchestration
  • 3 more changes...

Run 'autospec update' to upgrade
```

**Implementation:** `internal/cli/util/check.go:123` - Fetch changelog for latest version (requires GitHub API or embed).

**Design decision:** How to get changelog for unreleased versions?
- Option A: Embed at build → only shows changes up to current binary build
- Option B: Fetch from GitHub raw URL → requires network, but shows latest
- **Recommendation:** Option B for `ck`, Option A for `changelog` command

## Migration Plan

1. **Create `.autospec/changelog.yaml`** - Convert existing CHANGELOG.md to YAML format
2. **Build parser package** - `internal/changelog/` with tests
3. **Add generation tooling** - `.release/generate-changelog.go` + Makefile targets
4. **Update `/changelog` command** - Edit YAML instead of MD, run sync
5. **Update release workflow** - Replace bash script with Go extractor
6. **Add CLI command** - `autospec changelog` with embed support
7. **Integrate with `update`** - Show recent changes after upgrade
8. **Integrate with `ck`** - Preview highlights when update available
9. **Update CLAUDE.md** - Document YAML as source of truth

### 7. Slash Command Update: `.claude/commands/changelog.md`

Update the `/changelog` slash command to work with YAML source:

```markdown
## Task

Based on commits and guidelines:

1. **Group commits** into meaningful user-facing changes
2. **Draft entries** as YAML under `unreleased.changes`
3. **Update `.autospec/changelog.yaml`** (source of truth)
4. **Run `make changelog-sync`** to regenerate CHANGELOG.md
```

Current command edits CHANGELOG.md directly. New workflow:
- Drafts entries → adds to CHANGELOG.yaml → syncs to MD

### 8. Makefile Targets

```makefile
# Generate CHANGELOG.md from CHANGELOG.yaml
changelog-sync:
	go run .release/generate-changelog.go

# Validate CHANGELOG.md matches CHANGELOG.yaml (CI check)
changelog-check:
	go run .release/generate-changelog.go --check

# Shorthand for sync
changelog: changelog-sync
```

**Usage:**
- `make changelog-sync` - Regenerate MD after editing YAML
- `make changelog-check` - CI validation (fails if out of sync)

### 9. CLAUDE.md Updates

Add concise note to CLAUDE.md changelog section:

```markdown
## Changelog

**Source of truth:** `.autospec/changelog.yaml`

- Edit YAML, run `make changelog-sync` to update CHANGELOG.md
- Use `/changelog` command to draft entries from commits
- See `.release/CLAUDE.md` for entry writing guidelines
```

Replace any existing CHANGELOG.md direct-edit instructions.

## File References

| Existing File | Changes Needed |
|---------------|----------------|
| `internal/cli/util/update.go:151` | Add changelog display after update |
| `internal/cli/util/check.go:123` | Add changelog preview for available updates |
| `.github/workflows/release.yml:65-66` | Replace bash extraction with Go |
| `.release/extract-changelog.sh` | Deprecate/remove |
| `.release/CLAUDE.md` | Update with YAML workflow |
| `Makefile` | Add `changelog-sync` and `changelog-check` targets |
| `CLAUDE.md` | Add changelog source of truth note |
| `.claude/commands/changelog.md` | Update to edit YAML + sync |

## New Files

| File | Purpose |
|------|---------|
| `.autospec/changelog.yaml` | Source of truth |
| `internal/changelog/schema.go` | YAML schema structs |
| `internal/changelog/parse.go` | YAML loading |
| `internal/changelog/render.go` | Markdown generation |
| `internal/changelog/query.go` | Entry queries |
| `internal/changelog/changelog_test.go` | Unit tests |
| `internal/cli/util/changelog.go` | CLI command |
| `internal/cli/util/changelog_test.go` | CLI tests |
| `.release/generate-changelog.go` | MD generator tool |
| `.release/extract-changelog.go` | Release notes extractor |

## Build Embedding

```go
// internal/changelog/embed.go
//go:embed changelog.yaml
var embeddedChangelog []byte

func GetEmbedded() (*Changelog, error) {
    return Parse(embeddedChangelog)
}
```

**Note:** Requires copying `.autospec/changelog.yaml` to `internal/changelog/` at build time, or using relative embed path.

## Success Criteria

- [ ] CHANGELOG.yaml is the single edit point for changelog updates
- [ ] CHANGELOG.md is auto-generated and stays in sync
- [ ] GitHub releases use YAML for release descriptions
- [ ] `autospec changelog` shows entries without network
- [ ] `autospec update` shows what changed after upgrade
- [ ] `autospec ck` previews changes when update available
- [ ] Existing `.release/CLAUDE.md` guidance still applies (user-focused, grouped entries)
