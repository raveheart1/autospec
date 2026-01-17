# Task: Adversarial Review

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (Constitution Layer)

## Summary

Implement an adversarial review system where a second AI agent reviews implementation output for security vulnerabilities, complexity issues, code duplication, and other problems the implementing agent might miss. This creates a "red team / blue team" dynamic within the agentic workflow.

## Motivation

A single AI agent implementing code has blind spots—it optimizes for completing the task, not for catching its own mistakes. Adversarial review introduces a second perspective focused solely on finding problems, creating tension that improves code quality without human bottleneck.

## Configuration

### Config Toggle

```yaml
# .autospec/config.yml
verification:
  adversarial_review: true  # Enable/disable this feature
```

- **Config key**: `verification.adversarial_review`
- **Default**: `false`
- **Level presets**: Enabled at `full` level only
- **Override**: Can be explicitly set regardless of level

## Design

### Constitution Configuration

Add adversarial review settings to constitution constraints:

```yaml
# .autospec/memory/constitution.yaml
review:
  enabled: true
  mode: post-task          # post-task | post-phase | post-implement
  focus:
    - security             # OWASP Top 10, injection, auth bypass
    - complexity           # Hidden complexity, god functions
    - duplication          # Copy-paste code, DRY violations
    - error_handling       # Missing error paths, swallowed errors
    - resource_leaks       # Unclosed handles, memory leaks
  
  thresholds:
    max_issues_per_task: 5       # Fail if more issues found
    required_security_score: 8   # 1-10 scale
  
  reviewer_prompt_override: ""   # Custom prompt for reviewer agent
```

### Review Workflow

```
┌─────────────────────────────────────────────────────────────────┐
│                     ADVERSARIAL REVIEW FLOW                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Implementing Agent completes task                            │
│                        ↓                                         │
│  2. Reviewer Agent activated with:                               │
│     • Changed files                                              │
│     • Task context (requirements, acceptance criteria)           │
│     • Focus areas from constitution                              │
│                        ↓                                         │
│  3. Reviewer produces structured findings:                       │
│     • Issue severity (critical/high/medium/low)                  │
│     • Issue category (security/complexity/etc.)                  │
│     • Location (file:line)                                       │
│     • Description + suggested fix                                │
│                        ↓                                         │
│  4. Decision gate:                                               │
│     • No critical/high issues → Proceed                          │
│     • Issues found → Feed back to implementing agent             │
│                        ↓                                         │
│  5. Implementing agent addresses issues                          │
│     (Max retries apply)                                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Review Output Format

```yaml
# Structured review output
review_result:
  task_id: "TASK-001"
  files_reviewed:
    - internal/domain/cart.go
    - internal/domain/cart_test.go
  
  issues:
    - id: "REV-001"
      severity: high
      category: security
      file: internal/domain/cart.go
      line: 45
      title: "Potential integer overflow in total calculation"
      description: |
        The total calculation multiplies price * quantity without 
        overflow checking. For large quantities, this could wrap negative.
      suggestion: |
        Use checked arithmetic or validate quantity bounds:
        if quantity > MaxQuantity { return ErrQuantityTooLarge }
      
    - id: "REV-002"
      severity: medium
      category: complexity
      file: internal/domain/cart.go
      line: 78
      title: "Function exceeds complexity threshold"
      description: |
        The ProcessCheckout function has cyclomatic complexity of 15,
        exceeding the constitution limit of 10.
      suggestion: |
        Extract discount calculation and validation into separate functions.
  
  summary:
    total_issues: 2
    by_severity:
      critical: 0
      high: 1
      medium: 1
      low: 0
    security_score: 7
    recommendation: "address_before_merge"
```

### Command Interface

```bash
# Run adversarial review on a completed task
autospec review cart-feature --task TASK-001

# Review all tasks in a spec
autospec review cart-feature

# Review with specific focus override
autospec review cart-feature --focus security,error_handling

# Output as JSON for tooling
autospec review cart-feature --format json

# Dry-run (show what would be reviewed)
autospec review cart-feature --dry-run
```

### Integration with Implement

When `review.enabled: true` in constitution:

```bash
# Implement automatically triggers review after each task
autospec implement cart-feature
# → Task completes → Review runs → Issues fed back → Retry if needed

# Explicit flag to skip review (not recommended)
autospec implement cart-feature --skip-review

# Review-only mode (after manual implementation)
autospec implement cart-feature --review-only
```

## Implementation Notes

### Review Agent Prompt

The reviewer agent receives a specialized prompt:

```markdown
You are a code reviewer focused on finding problems, NOT on being helpful or polite.

Your job: Find issues the implementing agent missed.

Focus areas: {{focus_areas}}
Constitution rules: {{relevant_constitution_rules}}

Files to review:
{{changed_files}}

Task context:
{{task_description}}
{{acceptance_criteria}}

Output your findings in the structured format. Be specific about location 
and provide actionable suggestions. Do NOT comment on style or formatting
unless it hides bugs.

Severity guide:
- CRITICAL: Security vulnerability, data loss, crash
- HIGH: Logic error, requirement violation, resource leak
- MEDIUM: Complexity issue, maintainability concern
- LOW: Minor improvement opportunity
```

### Review Package

New package `internal/review/`:

- ReviewRunner: Orchestrates review session
- ReviewConfig: Loads focus areas and thresholds from constitution
- ReviewParser: Parses structured review output
- ReviewFeedback: Formats issues for implementing agent retry

### Session Isolation

Review runs in a separate Claude session to:
- Prevent context bleeding from implementation
- Keep reviewer "adversarial" (no shared context)
- Control costs (reviewer prompt is focused)

### Retry Integration

When review finds issues:

1. Issues formatted as structured feedback (see `05-structured-feedback.md`)
2. Implementing agent receives feedback with retry context
3. Retry count incremented
4. Process repeats until clean review or max retries

## Acceptance Criteria

1. `autospec review` command runs adversarial review on specified scope
2. Review uses separate agent session from implementation
3. Issues are structured and actionable
4. Integration with implement respects review.enabled config
5. Severity thresholds from constitution are enforced
6. Retry loop handles review feedback correctly
7. Review can be skipped with explicit flag
8. JSON output format available for tooling

## Security Considerations

The reviewer focuses on security by default when enabled. Key checks:

- Input validation gaps
- SQL/command injection vectors
- Authentication/authorization bypasses
- Sensitive data exposure
- CSRF/XSS vulnerabilities (for web code)
- Hardcoded secrets
- Insecure defaults

## Dependencies

- `01-verification-config.md` (respects verification level)
- `05-structured-feedback.md` (uses feedback format for retry)

## Estimated Scope

Medium-Large. New command, review runner, prompt engineering, implement integration.
