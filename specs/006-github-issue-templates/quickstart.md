# Quickstart: GitHub Issue Templates

**Feature**: 006-github-issue-templates
**Time to Complete**: 10 minutes
**Prerequisites**: Git repository on GitHub

## What You'll Build

Three GitHub issue templates that guide contributors to submit well-structured bug reports and feature requests:

1. **bug_report.md**: Template for reporting bugs with reproduction steps and environment details
2. **feature_request.md**: Template for suggesting features with problem statements and use cases
3. **config.yml**: Configuration to disable blank issues and provide contact links

## Step 1: Create Template Directory (1 min)

```bash
# From repository root
mkdir -p .github/ISSUE_TEMPLATE
```

**Why**: GitHub looks for issue templates in `.github/ISSUE_TEMPLATE/` directory.

## Step 2: Create Bug Report Template (3 min)

Create `.github/ISSUE_TEMPLATE/bug_report.md`:

```markdown
---
name: Bug Report
about: Report a defect or unexpected behavior
title: '[BUG] '
labels: bug, needs-triage
assignees: ''
---

## Bug Description

<!-- A clear and concise description of what the bug is -->

## Steps to Reproduce

<!-- Numbered steps to reproduce the behavior -->
1.
2.
3.

## Expected Behavior

<!-- What you expected to happen -->

## Actual Behavior

<!-- What actually happened -->

## Environment

- **OS**: <!-- e.g., Ubuntu 22.04, macOS 14.0, Windows 11 -->
- **Go version**: <!-- output of `go version` -->
- **autospec version**: <!-- output of `./autospec version` -->
- **Installation**: <!-- binary or source -->
- **Shell**: <!-- bash, zsh, fish, etc. -->

## Additional Context

<!-- Add screenshots, logs, related issues, or any other context about the problem -->
```

**Customize**:
- **labels**: Change to match your repository's labels
- **Environment section**: Adjust fields to match your project (language version, dependencies, etc.)
- **title prefix**: Use `[BUG]` or any prefix you prefer

## Step 3: Create Feature Request Template (3 min)

Create `.github/ISSUE_TEMPLATE/feature_request.md`:

```markdown
---
name: Feature Request
about: Suggest a new feature or enhancement
title: '[FEATURE] '
labels: enhancement, needs-discussion
assignees: ''
---

## Problem Statement

<!-- What problem does this solve? What need does it address? -->

## Use Case

<!-- Who would use this feature? How would they use it? What value does it provide? -->

## Proposed Solution

<!-- Describe one possible way to solve this problem -->

## Alternatives Considered

<!-- What other approaches did you consider? Why is the proposed solution better? -->

## Additional Context

<!-- Add mockups, examples from other projects, related issues, or any other context -->
```

**Customize**:
- **labels**: Match your repository's label scheme
- **sections**: Add/remove sections based on your needs (e.g., "Acceptance Criteria", "Technical Approach")

## Step 4: Create Configuration File (2 min)

Create `.github/ISSUE_TEMPLATE/config.yml`:

```yaml
blank_issues_enabled: false
contact_links:
  - name: Community Discussions
    url: https://github.com/YOUR-USERNAME/YOUR-REPO/discussions
    about: For questions, help, and general discussion
  - name: Documentation
    url: https://github.com/YOUR-USERNAME/YOUR-REPO/blob/main/README.md
    about: Read the documentation for setup and usage guides
```

**Customize**:
- **blank_issues_enabled**: Set to `true` if you want to allow blank issues
- **contact_links**: Update URLs to point to your repo's discussions, docs, website, etc.
- **Remove contact_links**: If you don't need them, delete the entire `contact_links:` section

## Step 5: Validate Templates (1 min)

```bash
# Check files exist
ls -la .github/ISSUE_TEMPLATE/

# Validate YAML syntax (if yq installed)
yq eval '.' .github/ISSUE_TEMPLATE/bug_report.md >/dev/null && echo "✓ bug_report.md valid"
yq eval '.' .github/ISSUE_TEMPLATE/feature_request.md >/dev/null && echo "✓ feature_request.md valid"
yq eval '.' .github/ISSUE_TEMPLATE/config.yml >/dev/null && echo "✓ config.yml valid"

# Or use Python if yq not available
python3 -c "import yaml; yaml.safe_load(open('.github/ISSUE_TEMPLATE/config.yml'))" && echo "✓ config.yml valid"
```

**Expected Output**:
```
✓ bug_report.md valid
✓ feature_request.md valid
✓ config.yml valid
```

## Step 6: Commit and Push (1 min)

```bash
git add .github/ISSUE_TEMPLATE/
git commit -m "Add GitHub issue templates for bug reports and feature requests"
git push origin main
```

## Step 7: Test on GitHub (2 min)

1. Navigate to your repository on GitHub
2. Click **"New Issue"** button
3. Verify you see the template chooser with:
   - **Bug Report** template
   - **Feature Request** template
   - **Contact links** (if configured)
   - **No blank issue option** (if disabled)
4. Select **Bug Report** template
5. Verify:
   - All sections populated correctly
   - Title shows `[BUG] ` prefix
   - Labels auto-selected (bug, needs-triage)
6. Repeat for **Feature Request** template

**Success Criteria**:
- ✅ Template chooser appears
- ✅ Both templates listed with correct names
- ✅ Templates populate issue form correctly
- ✅ Title prefixes and labels applied
- ✅ Blank issues disabled (if configured)
- ✅ Contact links appear (if configured)

## Common Issues

### Template chooser doesn't appear

**Cause**: Only one template exists and no config.yml

**Solution**: Either add a second template OR create config.yml (even empty file works)

### Templates have [object Object] or weird text

**Cause**: Invalid YAML frontmatter syntax

**Solution**: Validate YAML with `yq` or `yamllint`, check for:
- Mismatched quotes
- Missing colons
- Incorrect indentation
- Unclosed brackets/braces

### Labels don't appear with colors

**Cause**: Labels don't exist in repository yet

**Solution**: GitHub auto-creates labels on first use with random colors. To set colors:
1. Go to repo **Settings → Labels**
2. Edit the auto-created labels
3. Set colors and descriptions

### Blank issues still allowed

**Cause**: `blank_issues_enabled: true` in config.yml OR config.yml doesn't exist

**Solution**: Set `blank_issues_enabled: false` in config.yml and commit

### Contact links don't appear

**Cause**: Invalid contact_links syntax or relative URLs

**Solution**: Ensure each contact link has all three fields (name, url, about) and URLs are absolute (start with https://)

## Next Steps

**Enhance Templates**:
- Add more templates (e.g., question.md, documentation.md)
- Add more detailed guidance text in sections
- Include examples in template sections

**Improve Workflow**:
- Add automated YAML validation to CI/CD
- Create PR template (`.github/pull_request_template.md`)
- Set up issue labels and triage workflows

**Automate Validation**:
```bash
# Add to CI/CD pipeline
#!/bin/bash
for template in .github/ISSUE_TEMPLATE/*.md; do
  yq eval '.' "$template" >/dev/null || {
    echo "Invalid YAML in $template"
    exit 1
  }
done
echo "✓ All templates valid"
```

## Learn More

- [GitHub Docs: Issue Templates](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository)
- [GitHub Docs: Template Chooser](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/about-issue-and-pull-request-templates)
- [YAML Syntax](https://yaml.org/spec/1.2/spec.html)

## Troubleshooting

**Q: Can I use YAML forms instead of markdown templates?**

A: Yes, but they're more complex. See [GitHub's form schema docs](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-githubs-form-schema). Markdown templates are simpler and more flexible.

**Q: Can I enforce required fields?**

A: No, markdown templates can't enforce required fields. Contributors can delete any section. For enforcement, use YAML forms with `required: true` fields.

**Q: How do I preview templates before pushing?**

A: No official preview tool. Best practice: push to a test branch, create test issues, then merge to main when verified.

**Q: Can I have different templates for different branches?**

A: No, GitHub uses templates from the default branch (usually `main`). All branches see the same templates.

**Q: How do I migrate from old `.github/ISSUE_TEMPLATE.md` file?**

A: Rename to `.github/ISSUE_TEMPLATE/default.md` and add YAML frontmatter. Old single-file format is deprecated.

## Template Checklist

Before pushing templates, verify:

- [ ] All template files end in `.md`
- [ ] YAML frontmatter starts and ends with `---`
- [ ] Required fields present: `name`, `about`
- [ ] Title prefixes end with space (e.g., `'[BUG] '`)
- [ ] Labels exist in repository or are acceptable to auto-create
- [ ] config.yml has valid boolean for `blank_issues_enabled`
- [ ] Contact links have absolute URLs (https://)
- [ ] Section headings use `##` (H2)
- [ ] Guidance text is helpful but not overwhelming
- [ ] Templates tested on GitHub UI

## Success!

You've successfully added GitHub issue templates to your repository. Contributors will now see structured templates when creating issues, leading to better bug reports and feature requests with complete information.

**Time Saved**: Maintainers typically save 50% of time previously spent requesting additional information from contributors.

**Quality Improvement**: Structured templates lead to more complete initial reports, faster triage, and quicker resolution.
