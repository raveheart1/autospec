# Data Model: GitHub Issue Templates

**Feature**: 006-github-issue-templates
**Date**: 2025-10-23

## Overview

This feature uses static markdown files with YAML frontmatter. The "data model" describes the structure of template files and configuration, not traditional database entities.

## Entities

### Entity 1: Bug Report Template

**File**: `.github/ISSUE_TEMPLATE/bug_report.md`

**Structure**:
```markdown
---
name: Bug Report
about: Report a defect or unexpected behavior
title: '[BUG] '
labels: bug, needs-triage
assignees: ''
---

## Bug Description
[Clear description of the bug]

## Steps to Reproduce
1. [First step]
2. [Second step]
3. [...]

## Expected Behavior
[What should happen]

## Actual Behavior
[What actually happens]

## Environment
- OS: [e.g., Ubuntu 22.04, macOS 14.0, Windows 11]
- Go version: [output of `go version`]
- autospec version: [output of `./autospec version`]
- Installation: [binary/source]
- Shell: [bash/zsh/fish/etc.]

## Additional Context
[Screenshots, logs, related issues, etc.]
```

**Fields**:
- **name** (YAML): Template display name in GitHub chooser
- **about** (YAML): Brief description shown in chooser
- **title** (YAML): Default issue title prefix
- **labels** (YAML): Comma-separated labels to auto-apply
- **assignees** (YAML): GitHub usernames (empty for none)

**Sections** (Markdown):
1. Bug Description: Free-form description of problem
2. Steps to Reproduce: Numbered list of actions
3. Expected Behavior: What user expected to happen
4. Actual Behavior: What actually happened
5. Environment: System details (OS, versions, installation)
6. Additional Context: Optional supplementary information

**Validation Rules**:
- YAML frontmatter must be valid YAML syntax
- Required YAML fields: name, about
- All markdown sections should be present (guideline, not enforced)
- Title prefix should end with space for clean formatting

**State Transitions**: N/A (static file)

---

### Entity 2: Feature Request Template

**File**: `.github/ISSUE_TEMPLATE/feature_request.md`

**Structure**:
```markdown
---
name: Feature Request
about: Suggest a new feature or enhancement
title: '[FEATURE] '
labels: enhancement, needs-discussion
assignees: ''
---

## Problem Statement
[What problem does this solve? What need does it address?]

## Use Case
[Who would use this? How would they use it? What value does it provide?]

## Proposed Solution
[One possible way to solve this problem]

## Alternatives Considered
[What other approaches did you consider? Why is the proposed solution better?]

## Additional Context
[Mockups, examples from other projects, related issues, etc.]
```

**Fields**:
- **name** (YAML): Template display name
- **about** (YAML): Brief description
- **title** (YAML): Default title prefix
- **labels** (YAML): Auto-applied labels
- **assignees** (YAML): GitHub usernames (empty)

**Sections** (Markdown):
1. Problem Statement: Describes underlying need/pain point
2. Use Case: Who benefits, how they'd use it, value provided
3. Proposed Solution: One implementation approach
4. Alternatives Considered: Other options evaluated
5. Additional Context: Optional supporting materials

**Validation Rules**:
- YAML frontmatter must be valid
- Required YAML fields: name, about
- Problem Statement section should focus on "why" not "what"
- Proposed Solution should be one option, not a requirement

**State Transitions**: N/A (static file)

---

### Entity 3: Template Configuration

**File**: `.github/ISSUE_TEMPLATE/config.yml`

**Structure**:
```yaml
blank_issues_enabled: false
contact_links:
  - name: Community Support
    url: https://github.com/ariel-frischer/autospec/discussions
    about: For questions, help, and general discussion
  - name: Documentation
    url: https://github.com/ariel-frischer/autospec/blob/main/README.md
    about: Read the documentation for usage guides
```

**Fields**:
- **blank_issues_enabled** (boolean): Allow/disallow blank issues (no template)
- **contact_links** (array): List of external resources

**Contact Link Fields**:
- **name** (string): Display name for the link
- **url** (string): Full URL to external resource
- **about** (string): Brief description of what the link is for

**Validation Rules**:
- Must be valid YAML syntax
- `blank_issues_enabled` must be boolean (true/false)
- `contact_links` is optional but recommended
- Each contact_link must have name, url, and about
- URLs should be absolute (https://)

**State Transitions**: N/A (static file)

---

## Relationships

```
config.yml
    ↓ (configures behavior)
Template Chooser UI
    ↓ (user selects)
bug_report.md OR feature_request.md
    ↓ (populates)
New GitHub Issue
```

**Description**:
- config.yml controls whether blank issues are allowed and provides contact links
- User sees template chooser with bug_report and feature_request options
- Selecting a template populates new issue form with template content
- YAML frontmatter auto-applies labels and title prefix

---

## Data Flow

1. **Repository Configuration**:
   - Files committed to `.github/ISSUE_TEMPLATE/`
   - GitHub detects and parses templates
   - Template chooser becomes available

2. **Issue Creation**:
   - User clicks "New Issue" button
   - GitHub displays template chooser (if multiple templates or config.yml exists)
   - User selects bug_report or feature_request
   - GitHub populates issue form with template markdown
   - Title prefix and labels auto-applied from YAML frontmatter

3. **Issue Submission**:
   - User fills out sections (can modify/delete)
   - User clicks "Submit new issue"
   - Issue created with labels, title, and content

---

## Constraints

1. **GitHub Format Requirements**:
   - YAML frontmatter must be between `---` delimiters
   - File must start with YAML frontmatter
   - Markdown content follows frontmatter

2. **File Location**:
   - Templates must be in `.github/ISSUE_TEMPLATE/` directory
   - Filenames can be anything (descriptive names recommended)
   - config.yml must be named exactly `config.yml`

3. **Template Limitations**:
   - Cannot enforce required fields (users can delete sections)
   - Cannot validate input (no forms, just markdown)
   - Cannot conditionally show/hide sections

4. **Label Requirements**:
   - Labels referenced in templates should exist in repository
   - Non-existent labels will be created when first issue is submitted
   - Labels are comma-separated in YAML

---

## Testing Validation

**Automated Tests**:
```bash
# YAML syntax validation
yq eval '.github/ISSUE_TEMPLATE/bug_report.md' > /dev/null
yq eval '.github/ISSUE_TEMPLATE/feature_request.md' > /dev/null
yq eval '.github/ISSUE_TEMPLATE/config.yml' > /dev/null

# File existence
test -f .github/ISSUE_TEMPLATE/bug_report.md
test -f .github/ISSUE_TEMPLATE/feature_request.md
test -f .github/ISSUE_TEMPLATE/config.yml

# Required sections present
grep -q "## Bug Description" .github/ISSUE_TEMPLATE/bug_report.md
grep -q "## Steps to Reproduce" .github/ISSUE_TEMPLATE/bug_report.md
grep -q "## Environment" .github/ISSUE_TEMPLATE/bug_report.md

grep -q "## Problem Statement" .github/ISSUE_TEMPLATE/feature_request.md
grep -q "## Use Case" .github/ISSUE_TEMPLATE/feature_request.md
```

**Manual Tests**:
1. Navigate to repository on GitHub
2. Click "New Issue"
3. Verify template chooser appears with both templates
4. Select bug_report template
5. Verify all sections populated correctly
6. Verify title prefix `[BUG] ` present
7. Verify labels `bug, needs-triage` auto-applied
8. Repeat for feature_request template
9. Verify blank issues are disabled (no "Open a blank issue" option)

---

## Implementation Notes

**YAML Parsing Tools**:
- Use `yq` for validation in CI/CD
- Fallback to Python's `yaml.safe_load()` if yq unavailable
- GitHub performs its own parsing (official validator)

**Section Guidance**:
- Use HTML comments `<!-- guidance text -->` for section hints
- Or italic text: `*Describe the problem in your own words*`
- Keep guidance minimal - respect contributor's time

**Label Management**:
- Coordinate with repository maintainers on label names
- Use existing labels: `bug`, `enhancement`, `needs-triage`, `needs-discussion`
- If labels don't exist, GitHub creates them on first use
