# Contract: Template Validation API

**Version**: 1.0.0
**Date**: 2025-10-23
**Purpose**: Define bash functions for validating GitHub issue templates

## Overview

This contract specifies the validation functions used to verify issue template correctness. All functions follow bash best practices and return standard exit codes.

## Function Contracts

### validate_yaml_syntax()

**Purpose**: Validate YAML frontmatter syntax in template file

**Signature**:
```bash
validate_yaml_syntax <file_path>
```

**Parameters**:
- `file_path`: Absolute or relative path to template file

**Return Codes**:
- `0`: YAML syntax is valid
- `1`: YAML syntax is invalid or file not found
- `2`: Validation tool not available

**Behavior**:
```bash
validate_yaml_syntax() {
  local file="$1"

  # Check file exists
  if [[ ! -f "$file" ]]; then
    echo "Error: File not found: $file" >&2
    return 1
  fi

  # Try yq first
  if command -v yq >/dev/null 2>&1; then
    if yq eval '.' "$file" >/dev/null 2>&1; then
      return 0
    else
      echo "Error: Invalid YAML in $file" >&2
      return 1
    fi
  fi

  # Fallback to Python yaml module
  if command -v python3 >/dev/null 2>&1; then
    if python3 -c "import yaml; yaml.safe_load(open('$file'))" >/dev/null 2>&1; then
      return 0
    else
      echo "Error: Invalid YAML in $file" >&2
      return 1
    fi
  fi

  echo "Error: No YAML validation tool available (yq or python3)" >&2
  return 2
}
```

**Usage Example**:
```bash
if validate_yaml_syntax .github/ISSUE_TEMPLATE/bug_report.md; then
  echo "✓ YAML syntax valid"
else
  echo "✗ YAML syntax invalid"
  exit 1
fi
```

---

### validate_required_fields()

**Purpose**: Verify required YAML frontmatter fields are present

**Signature**:
```bash
validate_required_fields <file_path>
```

**Parameters**:
- `file_path`: Path to template file

**Required Fields**:
- `name`: Template display name
- `about`: Template description

**Return Codes**:
- `0`: All required fields present
- `1`: Missing required fields or validation failed

**Behavior**:
```bash
validate_required_fields() {
  local file="$1"
  local missing_fields=()

  # Check 'name' field
  if ! yq eval '.name' "$file" 2>/dev/null | grep -qv '^null$'; then
    missing_fields+=("name")
  fi

  # Check 'about' field
  if ! yq eval '.about' "$file" 2>/dev/null | grep -qv '^null$'; then
    missing_fields+=("about")
  fi

  if [[ ${#missing_fields[@]} -gt 0 ]]; then
    echo "Error: Missing required fields in $file: ${missing_fields[*]}" >&2
    return 1
  fi

  return 0
}
```

**Usage Example**:
```bash
if validate_required_fields .github/ISSUE_TEMPLATE/bug_report.md; then
  echo "✓ Required fields present"
else
  echo "✗ Missing required fields"
  exit 1
fi
```

---

### validate_template_sections()

**Purpose**: Verify required markdown sections are present

**Signature**:
```bash
validate_template_sections <file_path> <template_type>
```

**Parameters**:
- `file_path`: Path to template file
- `template_type`: Either "bug_report" or "feature_request"

**Required Sections**:

**Bug Report**:
- `## Bug Description`
- `## Steps to Reproduce`
- `## Expected Behavior`
- `## Actual Behavior`
- `## Environment`

**Feature Request**:
- `## Problem Statement`
- `## Use Case`
- `## Proposed Solution`

**Return Codes**:
- `0`: All required sections present
- `1`: Missing sections
- `3`: Invalid template type

**Behavior**:
```bash
validate_template_sections() {
  local file="$1"
  local type="$2"
  local missing_sections=()

  case "$type" in
    bug_report)
      local required=(
        "## Bug Description"
        "## Steps to Reproduce"
        "## Expected Behavior"
        "## Actual Behavior"
        "## Environment"
      )
      ;;
    feature_request)
      local required=(
        "## Problem Statement"
        "## Use Case"
        "## Proposed Solution"
      )
      ;;
    *)
      echo "Error: Invalid template type: $type" >&2
      return 3
      ;;
  esac

  for section in "${required[@]}"; do
    if ! grep -qF "$section" "$file"; then
      missing_sections+=("$section")
    fi
  done

  if [[ ${#missing_sections[@]} -gt 0 ]]; then
    echo "Error: Missing sections in $file:" >&2
    printf '  - %s\n' "${missing_sections[@]}" >&2
    return 1
  fi

  return 0
}
```

**Usage Example**:
```bash
if validate_template_sections .github/ISSUE_TEMPLATE/bug_report.md bug_report; then
  echo "✓ All sections present"
else
  echo "✗ Missing sections"
  exit 1
fi
```

---

### validate_config_file()

**Purpose**: Validate config.yml structure and syntax

**Signature**:
```bash
validate_config_file <file_path>
```

**Parameters**:
- `file_path`: Path to config.yml

**Validated Fields**:
- `blank_issues_enabled`: Must be boolean (true/false)
- `contact_links`: If present, each must have name, url, about

**Return Codes**:
- `0`: Config file is valid
- `1`: Config file is invalid
- `2`: File not found (acceptable - config is optional)

**Behavior**:
```bash
validate_config_file() {
  local file="$1"

  # Config file is optional
  if [[ ! -f "$file" ]]; then
    return 2
  fi

  # Validate YAML syntax first
  if ! validate_yaml_syntax "$file"; then
    return 1
  fi

  # Validate blank_issues_enabled if present
  local blank_issues
  blank_issues=$(yq eval '.blank_issues_enabled' "$file" 2>/dev/null)
  if [[ "$blank_issues" != "null" && "$blank_issues" != "true" && "$blank_issues" != "false" ]]; then
    echo "Error: blank_issues_enabled must be boolean (true/false)" >&2
    return 1
  fi

  # Validate contact_links if present
  local links_count
  links_count=$(yq eval '.contact_links | length' "$file" 2>/dev/null)
  if [[ "$links_count" != "null" && "$links_count" -gt 0 ]]; then
    for ((i=0; i<links_count; i++)); do
      local name about url
      name=$(yq eval ".contact_links[$i].name" "$file" 2>/dev/null)
      about=$(yq eval ".contact_links[$i].about" "$file" 2>/dev/null)
      url=$(yq eval ".contact_links[$i].url" "$file" 2>/dev/null)

      if [[ "$name" == "null" || "$about" == "null" || "$url" == "null" ]]; then
        echo "Error: contact_links[$i] missing required fields (name, url, about)" >&2
        return 1
      fi
    done
  fi

  return 0
}
```

**Usage Example**:
```bash
result=0
validate_config_file .github/ISSUE_TEMPLATE/config.yml
case $? in
  0) echo "✓ Config file valid" ;;
  2) echo "ℹ Config file not found (optional)" ;;
  *) echo "✗ Config file invalid"; result=1 ;;
esac
exit $result
```

---

### validate_all_templates()

**Purpose**: Run all validations on all template files

**Signature**:
```bash
validate_all_templates [template_dir]
```

**Parameters**:
- `template_dir`: Path to ISSUE_TEMPLATE directory (default: `.github/ISSUE_TEMPLATE`)

**Return Codes**:
- `0`: All validations passed
- `1`: One or more validations failed
- `4`: Template directory not found

**Behavior**:
```bash
validate_all_templates() {
  local dir="${1:-.github/ISSUE_TEMPLATE}"
  local failed=0

  if [[ ! -d "$dir" ]]; then
    echo "Error: Template directory not found: $dir" >&2
    return 4
  fi

  # Validate config.yml
  echo "Validating config.yml..."
  case $(validate_config_file "$dir/config.yml"; echo $?) in
    0) echo "  ✓ Valid" ;;
    2) echo "  ℹ Not found (optional)" ;;
    *) echo "  ✗ Invalid"; ((failed++)) ;;
  esac

  # Validate bug_report.md
  if [[ -f "$dir/bug_report.md" ]]; then
    echo "Validating bug_report.md..."
    if validate_yaml_syntax "$dir/bug_report.md" && \
       validate_required_fields "$dir/bug_report.md" && \
       validate_template_sections "$dir/bug_report.md" bug_report; then
      echo "  ✓ Valid"
    else
      echo "  ✗ Invalid"
      ((failed++))
    fi
  fi

  # Validate feature_request.md
  if [[ -f "$dir/feature_request.md" ]]; then
    echo "Validating feature_request.md..."
    if validate_yaml_syntax "$dir/feature_request.md" && \
       validate_required_fields "$dir/feature_request.md" && \
       validate_template_sections "$dir/feature_request.md" feature_request; then
      echo "  ✓ Valid"
    else
      echo "  ✗ Invalid"
      ((failed++))
    fi
  fi

  if [[ $failed -eq 0 ]]; then
    echo "✓ All templates valid"
    return 0
  else
    echo "✗ $failed template(s) failed validation"
    return 1
  fi
}
```

**Usage Example**:
```bash
# Validate templates in default location
if validate_all_templates; then
  echo "Ready to commit"
else
  echo "Fix validation errors before committing"
  exit 1
fi

# Validate templates in custom location
validate_all_templates /path/to/templates
```

---

## Integration with Test Framework

### Bats Test Suite Structure

**File**: `tests/github_templates/validation_test.bats`

```bash
#!/usr/bin/env bats

# Load validation library
load '../test_helper'
source ./path/to/validation_lib.sh

setup() {
  # Create temp directory for test fixtures
  TEST_DIR=$(mktemp -d)
  mkdir -p "$TEST_DIR/.github/ISSUE_TEMPLATE"
}

teardown() {
  # Clean up temp directory
  rm -rf "$TEST_DIR"
}

@test "validate_yaml_syntax: valid YAML returns 0" {
  cat > "$TEST_DIR/test.md" <<'EOF'
---
name: Test
about: Test template
---
# Content
EOF

  run validate_yaml_syntax "$TEST_DIR/test.md"
  [ "$status" -eq 0 ]
}

@test "validate_yaml_syntax: invalid YAML returns 1" {
  cat > "$TEST_DIR/test.md" <<'EOF'
---
name: Test
about: [unclosed array
---
EOF

  run validate_yaml_syntax "$TEST_DIR/test.md"
  [ "$status" -eq 1 ]
}

@test "validate_required_fields: missing name returns 1" {
  cat > "$TEST_DIR/test.md" <<'EOF'
---
about: Test template
---
EOF

  run validate_required_fields "$TEST_DIR/test.md"
  [ "$status" -eq 1 ]
  [[ "$output" =~ "name" ]]
}

# ... more tests
```

---

## Performance Requirements

All validation functions MUST complete in <100ms for typical template files:

| Function | Max Duration | Typical File Size |
|----------|--------------|-------------------|
| validate_yaml_syntax() | 50ms | <5KB |
| validate_required_fields() | 20ms | <5KB |
| validate_template_sections() | 30ms | <10KB |
| validate_config_file() | 50ms | <2KB |
| validate_all_templates() | 200ms | 3-5 files |

**Optimization Notes**:
- Use `grep -qF` for fixed string matching (faster than regex)
- Avoid repeated file reads - cache content if needed
- Use `yq` over Python when available (faster startup)

---

## Error Message Format

All validation functions MUST output errors to stderr with this format:

```
Error: <brief description>
  - <detail 1>
  - <detail 2>
```

**Examples**:
```bash
Error: Missing required fields in bug_report.md: name about
Error: Missing sections in feature_request.md:
  - ## Problem Statement
  - ## Use Case
Error: contact_links[0] missing required fields (name, url, about)
```

**Rationale**: Consistent error format enables:
- Easy parsing by CI/CD tools
- Clear debugging for developers
- Structured error reporting

---

## Exit Code Standards

All validation functions follow this exit code convention:

| Code | Meaning | Action |
|------|---------|--------|
| 0 | Success | Continue |
| 1 | Validation failed | Fix errors and retry |
| 2 | Optional file missing | Continue (not an error) |
| 3 | Invalid arguments | Fix function call |
| 4 | Required file/dir missing | Create file/dir |

**Usage in CI/CD**:
```bash
#!/bin/bash
set -e

validate_all_templates || exit 1
echo "✓ Templates validated, proceeding with deployment"
```

---

## Dependencies

**Required Tools**:
- `bash` 4.0+ (for arrays and modern syntax)
- `grep` (for section matching)
- One of: `yq` (preferred) or `python3` with `yaml` module

**Optional Tools**:
- `yamllint`: More strict YAML validation
- `shellcheck`: Validate validation scripts themselves

**Dependency Check**:
```bash
check_dependencies() {
  local missing=()

  if ! command -v yq >/dev/null 2>&1 && \
     ! command -v python3 >/dev/null 2>&1; then
    missing+=("yq or python3")
  fi

  if ! command -v grep >/dev/null 2>&1; then
    missing+=("grep")
  fi

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "Error: Missing required dependencies: ${missing[*]}" >&2
    return 1
  fi

  return 0
}
```

---

## Testing the Validators

**Meta-Testing**: Validators themselves must be tested

```bash
# Test that validator catches invalid YAML
@test "validator detects invalid YAML" {
  echo "---" > test.md
  echo "invalid: [unclosed" >> test.md
  echo "---" >> test.md

  run validate_yaml_syntax test.md
  [ "$status" -eq 1 ]
}

# Test that validator allows valid templates
@test "validator accepts valid template" {
  cat > test.md <<'EOF'
---
name: Valid Template
about: This is valid
---
## Section 1
Content
EOF

  run validate_yaml_syntax test.md
  [ "$status" -eq 0 ]
}
```

---

## Future Enhancements

**Potential Additions** (not in current scope):
- Label existence validation (check labels exist in repo)
- URL validation in contact_links (HTTP HEAD request)
- Markdown linting (using markdownlint)
- Spell checking in templates
- Template content quality checks (section length, clarity)

**Backward Compatibility**: Any new validators MUST:
- Use separate functions (don't modify existing)
- Return standard exit codes
- Follow error message format
- Document performance requirements
