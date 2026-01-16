# Contract: GitHub Issue Template Format

**Version**: 1.0.0
**Date**: 2025-10-23
**Authority**: GitHub Issue Template Specification

## Overview

This contract defines the expected format for GitHub issue templates (markdown templates with YAML frontmatter) and configuration files.

## Template File Format Contract

### YAML Frontmatter Structure

**Format**: YAML between triple-dash delimiters

```yaml
---
name: string (required)
about: string (required)
title: string (optional)
labels: string (optional, comma-separated)
assignees: string (optional, comma-separated)
---
```

**Field Specifications**:

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| name | string | Yes | Template name shown in chooser | "Bug Report" |
| about | string | Yes | Brief description for chooser | "Report a defect or unexpected behavior" |
| title | string | No | Default issue title or prefix | "[BUG] " |
| labels | string | No | Comma-separated label names | "bug, needs-triage" |
| assignees | string | No | Comma-separated GitHub usernames | "username1, username2" |

**Constraints**:
- Frontmatter MUST start and end with exactly `---` (three dashes)
- Frontmatter MUST be at the beginning of the file
- YAML MUST be valid according to YAML 1.2 spec
- String values MAY be quoted or unquoted
- Empty strings are allowed (e.g., `assignees: ''`)

**Invalid Examples**:
```yaml
# Missing closing delimiter - INVALID
---
name: Bug Report
about: Report a bug

# Missing required fields - INVALID
---
title: Bug Report
---

# Frontmatter not at beginning - INVALID
Some text
---
name: Bug Report
about: Description
---
```

### Markdown Body Structure

**Format**: Standard markdown following YAML frontmatter

**Requirements**:
- Body MUST follow YAML frontmatter (after closing `---`)
- Body MAY include any valid markdown syntax
- Headings, lists, code blocks, emphasis are supported
- HTML comments are allowed for guidance text

**Recommended Structure**:
```markdown
---
[YAML frontmatter]
---

## Section 1 Name
[Description or guidance]

## Section 2 Name
[Description or guidance]

...
```

**Best Practices**:
- Use `##` (H2) for main sections
- Provide guidance text in brackets or HTML comments
- Keep total length under 100 lines for readability
- Use consistent formatting across templates

## Configuration File Contract

### config.yml Format

**File**: `.github/ISSUE_TEMPLATE/config.yml`

**Structure**:
```yaml
blank_issues_enabled: boolean
contact_links:
  - name: string
    url: string
    about: string
  - name: string
    url: string
    about: string
```

**Field Specifications**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| blank_issues_enabled | boolean | No (default: true) | Allow issues without templates |
| contact_links | array | No | External resources for non-issue discussions |
| contact_links[].name | string | Yes (if contact_links present) | Link display name |
| contact_links[].url | string | Yes (if contact_links present) | Absolute URL to resource |
| contact_links[].about | string | Yes (if contact_links present) | Brief description of resource |

**Constraints**:
- `blank_issues_enabled` MUST be boolean: `true` or `false`
- `contact_links` is OPTIONAL but if present MUST be an array
- Each contact link MUST have all three fields (name, url, about)
- URLs SHOULD be absolute (starting with http:// or https://)
- Empty config file is valid (uses GitHub defaults)

**Valid Examples**:
```yaml
# Minimal config
blank_issues_enabled: false

# Full config
blank_issues_enabled: false
contact_links:
  - name: Community Support
    url: https://github.com/owner/repo/discussions
    about: For questions and general discussion
  - name: Documentation
    url: https://example.com/docs
    about: Read the documentation
```

**Invalid Examples**:
```yaml
# Boolean as string - INVALID
blank_issues_enabled: "false"

# Missing required fields in contact_links - INVALID
contact_links:
  - name: Docs
    url: https://example.com

# Relative URL - DISCOURAGED (may not work)
contact_links:
  - name: Docs
    url: /docs
    about: Documentation
```

## File Location Contract

### Required Paths

**Directory**: `.github/ISSUE_TEMPLATE/`

**File Naming**:
- Template files: Any filename ending in `.md` (descriptive names recommended)
- Configuration: MUST be named `config.yml` (exact name required)

**Examples**:
```
✅ VALID:
.github/ISSUE_TEMPLATE/bug_report.md
.github/ISSUE_TEMPLATE/feature_request.md
.github/ISSUE_TEMPLATE/config.yml

✅ ALSO VALID:
.github/ISSUE_TEMPLATE/01-bug-report.md
.github/ISSUE_TEMPLATE/02-feature-request.md

❌ INVALID:
.github/bug_report.md              # Wrong directory
.github/ISSUE_TEMPLATE/config.yaml # Wrong extension
issue_templates/bug_report.md      # Wrong directory name
```

**Case Sensitivity**:
- Directory name IS case-sensitive: `.github/ISSUE_TEMPLATE/` (uppercase ISSUE_TEMPLATE)
- Alternative form: `.github/issue_template/` (lowercase, deprecated but may work)

## GitHub Behavior Contract

### Template Discovery

**When GitHub Shows Template Chooser**:
1. Multiple template files exist in `.github/ISSUE_TEMPLATE/`
2. OR config.yml exists in `.github/ISSUE_TEMPLATE/`
3. User clicks "New Issue" button

**When Blank Issues Allowed**:
- `blank_issues_enabled: true` in config.yml
- OR config.yml doesn't exist
- User sees "Open a blank issue" option

**When Blank Issues Disabled**:
- `blank_issues_enabled: false` in config.yml
- User MUST select a template to create issue

### Template Application

**Auto-Applied Fields (from YAML frontmatter)**:
- **title**: Populates issue title input
- **labels**: Labels auto-selected in label dropdown
- **assignees**: Assignees pre-selected

**User Can Override**:
- Title can be edited
- Labels can be added/removed
- Assignees can be changed
- Markdown body can be modified/deleted

**No Enforcement**:
- GitHub does NOT enforce required fields
- Users CAN delete all template content
- Users CAN submit empty issues (if blank issues enabled)

### Label Behavior

**If Label Exists**:
- Label applied from template frontmatter
- Label appears with correct color and description

**If Label Doesn't Exist**:
- GitHub creates label on first use
- Label gets random color
- No description set (can be edited later)

**Best Practice**: Create labels before referencing in templates

## Validation Contract

### Syntax Validation

**YAML Syntax**:
```bash
# Valid YAML check
yq eval '.github/ISSUE_TEMPLATE/template.md' > /dev/null
# Exit 0: valid, Exit 1: invalid
```

**Frontmatter Extraction**:
```bash
# Extract frontmatter (between first two --- delimiters)
sed -n '/^---$/,/^---$/p' template.md | sed '1d;$d'
```

### Required Fields Validation

**Check Required Fields**:
```bash
# Validate 'name' field present
yq eval '.name' template.md | grep -qv '^null$'

# Validate 'about' field present
yq eval '.about' template.md | grep -qv '^null$'
```

**Exit Codes**:
- 0: Valid (required fields present)
- 1: Invalid (missing required fields)

### Section Validation

**Check Required Sections**:
```bash
# Bug report sections
grep -q "## Bug Description" bug_report.md
grep -q "## Steps to Reproduce" bug_report.md
grep -q "## Expected Behavior" bug_report.md
grep -q "## Actual Behavior" bug_report.md
grep -q "## Environment" bug_report.md

# Feature request sections
grep -q "## Problem Statement" feature_request.md
grep -q "## Use Case" feature_request.md
grep -q "## Proposed Solution" feature_request.md
```

**Exit Codes**:
- 0: Section found
- 1: Section missing

## Error Handling Contract

### Template Parsing Errors

**Invalid YAML Frontmatter**:
- **GitHub Behavior**: Template ignored, not shown in chooser
- **No Error Message**: GitHub silently skips invalid templates
- **User Impact**: Template unavailable

**Missing Required Fields**:
- **GitHub Behavior**: Template may still appear but with incomplete info
- **Fallback**: Uses filename as display name if `name` missing

**Malformed Markdown**:
- **GitHub Behavior**: Renders as best as possible
- **Broken Formatting**: Users see raw markdown or broken formatting

### Configuration Errors

**Invalid config.yml**:
- **GitHub Behavior**: Falls back to default behavior
- **Default**: blank_issues_enabled = true, no contact links

**Invalid Contact Links**:
- **GitHub Behavior**: Skips invalid entries, shows valid ones
- **No Error**: Invalid entries silently ignored

## Testing Contract

### Automated Tests

**Minimum Test Coverage**:
```bash
#!/usr/bin/env bats

# File existence tests
@test "bug_report.md exists" {
  [ -f .github/ISSUE_TEMPLATE/bug_report.md ]
}

# YAML syntax tests
@test "bug_report.md has valid YAML frontmatter" {
  yq eval '.name' .github/ISSUE_TEMPLATE/bug_report.md >/dev/null
}

# Required fields tests
@test "bug_report.md has name field" {
  result=$(yq eval '.name' .github/ISSUE_TEMPLATE/bug_report.md)
  [ "$result" != "null" ]
}

# Section presence tests
@test "bug_report.md has all required sections" {
  grep -q "## Bug Description" .github/ISSUE_TEMPLATE/bug_report.md
  grep -q "## Steps to Reproduce" .github/ISSUE_TEMPLATE/bug_report.md
  grep -q "## Environment" .github/ISSUE_TEMPLATE/bug_report.md
}
```

### Manual Tests

**GitHub UI Verification**:
1. Push templates to GitHub repository
2. Navigate to repository
3. Click "New Issue"
4. Verify template chooser displays
5. Select each template and verify:
   - Template content populates correctly
   - Title prefix applied
   - Labels auto-selected
   - All sections present and readable
6. Verify blank issues disabled (if configured)
7. Verify contact links appear (if configured)

**Acceptance Criteria**:
- ✅ Template chooser appears
- ✅ All templates listed with correct names
- ✅ Template selection populates issue form
- ✅ YAML fields applied (title, labels)
- ✅ Markdown renders correctly
- ✅ Config.yml settings respected

## Version Compatibility

**GitHub Support**:
- Issue templates: Supported since ~2018
- Multiple templates: Requires template chooser (2018+)
- config.yml: Supported since ~2019

**Browser Compatibility**:
- Works in all modern browsers (Chrome, Firefox, Safari, Edge)
- Mobile browsers supported (responsive UI)

**API Support**:
- Templates apply only through web UI
- GitHub API issue creation ignores templates
- CLI tools (`gh` command) may or may not use templates

## References

- [GitHub Docs: Issue Templates](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository)
- [GitHub Docs: Template Syntax](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-githubs-form-schema)
- [YAML 1.2 Specification](https://yaml.org/spec/1.2/spec.html)
