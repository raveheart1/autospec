# Autospec DAG Improvement Tasks

## Critical Requirements (Must Have Before Full Release)

### Pre-Execution Spec Folder Validation (CRITICAL)
**Rule:** Before `dag run` executes, ALL spec folders defined in dag.yaml MUST already exist.

**Why:** Prevents conflicts with other specs on the main worktree repo. Ensures clean isolation.

**UX Decision: Auto-Prepare with Escape Hatches**

Default: `dag run` auto-prepares if needed, with visibility. Flags control behavior for agents.

| Flag | Behavior |
|------|----------|
| (none) | Interactive: "3 folders missing. Create them? [Y/n]" |
| `--yes` / `--non-interactive` | Auto-create without prompts, proceed (for agents) |
| `--prepare-only` | Create folders, exit 0 (CI check mode) |
| `--no-prepare` | Fail fast if folders missing (safety mode) |

**Requirements:**
- [ ] `dag run` validates all spec folders exist before starting
- [ ] If folders missing → show what's needed, offer to create (interactive) or auto-create (with `--yes`)
- [ ] Spec naming/numbering must be consistent (e.g., `specs/001-dag-spec-1`)
- [ ] `dag prepare` available as separate command for review-only workflow
- [ ] Folder naming scheme: `specs/<NNN>-<dag-id>-<feature-id>/`

**Interactive Example:**
```
$ autospec dag run my-app.yaml
⚠️  3 of 5 spec folders missing. Creating them...
   specs/003-myapp-auth/
   specs/004-myapp-api/
   specs/005-myapp-ui/

Run `dag prepare my-app.yaml` to review before continuing.
Press Enter to continue, Ctrl+C to cancel
```

**Non-Interactive Example:**
```
$ autospec dag run my-app.yaml --yes
ℹ️  Creating 3 missing spec folders... done
ℹ️  Starting DAG execution...
```

### User Observability & Simplicity (CORE PRINCIPLE)
Every feature must answer: "How does this make the user's life easier?"

**Guidelines:**
- Default to simple, add complexity only when needed
- Show progress clearly: what's running, what's waiting, what's done
- Explain WHY something is happening (not just WHAT)
- Provide clear next steps when things go wrong
- Design for the "happy path" but handle errors gracefully

### Worktree State Tracking
**Problem:** Hard to know which worktree belongs to which spec, what's in it, if it's dirty.

**Requirements:**
- [ ] `dag status` shows worktree → spec mapping
- [ ] Track worktree state: clean, dirty, committed, merged
- [ ] Show which files are uncommitted in each worktree
- [ ] Warn if worktree has uncommitted changes before operations
- [ ] `dag cleanup` shows what will be deleted before doing it

---

## Active Ideas

### 1. Improve Parallel Execution UX
**Problem:** Users must manually add `--parallel` flags, which isn't intuitive.

**Ideas:**
- Add interactive prompts when running `dag run` to ask about parallel execution
- Add `--skip-parallel-prompts` flag to skip these prompts for CI/automation
- Smart defaults based on DAG structure (auto-parallel if no cross-spec dependencies)
- Show execution plan preview with parallel vs sequential visualization

**Acceptance Criteria:**
- [ ] `dag run` without flags prompts user for parallel/sequential preference
- [ ] `--skip-parallel-prompts` uses config default or sequential fallback
- [ ] `--yes` or `--non-interactive` skips all prompts
- [ ] Preview shows which specs will run in parallel vs sequential

---

### 2. Improve Worktree Handling
**Problem:** Current worktree handling has rough edges in practice.

**Ideas:**
- Better worktree cleanup on interruption/failure
- Option to reuse worktrees for faster re-runs
- Worktree health checks before execution
- Clearer worktree naming/organization
- Show worktree disk usage and auto-cleanup old ones

**Acceptance Criteria:**
- [ ] Worktrees are properly cleaned up on SIGINT/SIGTERM
- [ ] `--reuse-worktrees` flag for faster re-runs (when specs unchanged)
- [ ] `dag worktree status` command to show all DAG-related worktrees
- [ ] Auto-cleanup worktrees older than N days (configurable)

---

### 3. Improve DAG YAML Schema
**Problem:** Current schema may be limiting for complex workflows.

**Ideas:**
- Add metadata fields (author, created, tags, estimated duration)
- Support for conditional execution (run this spec only if X)
- Resource requirements per spec (CPU, memory hints)
- Retry policies at spec level
- Pre/post hooks per spec
- Global environment variables section

**Acceptance Criteria:**
- [ ] Schema validation for new fields
- [ ] Backward compatibility with existing dag.yaml files
- [ ] Documentation for all schema fields
- [ ] Example dag.yaml files for common patterns

---

### 4. Spec Generation Workflow (Pre-execution)
**Problem:** Specs can exist before running DAG or not exist — inconsistent experience.

**Goal:** Unified workflow that generates all specs upfront, then runs them.

**Proposed Flow:**
1. Parse dag.yaml
2. Check which specs/<id>/ folders exist
3. For missing specs: generate spec.yaml from dag feature description
4. Run "spec review" phase — user reviews all generated specs
5. After approval: execute DAG

**Ideas:**
- `dag prepare` command — generate all missing specs without running
- `dag review` command — interactive review of all specs before execution
- Auto-generate spec.yaml from feature.description + layer context
- Show progress: "Generating 5 specs... 3 exist, 2 new"

**Acceptance Criteria:**
- [ ] `dag prepare` generates all missing spec folders and spec.yaml files
- [ ] `dag review` opens each spec.yaml in $EDITOR for review
- [ ] `dag run --prepare` does prepare + run in one command
- [ ] Generated specs include layer context and dependencies
- [ ] Spec generation uses AI to expand feature.description into full spec

---

### 5. Post-Spec Integration Agent
**Problem:** Specs run in isolation but real applications need integration.

**Goal:** Agentic flow that handles spec-to-spec integration.

**When It Runs:**
- After each spec completes (within a layer)
- After each layer completes (cross-layer integration)
- On-demand via `dag integrate` command

**Responsibilities:**
- Check if spec A's changes break spec B's assumptions
- Generate/update integration tests between specs
- Handle shared types/interfaces/contracts
- Update API clients when server changes
- Propagate changes across dependent specs

**Acceptance Criteria:**
- [ ] Integration agent runs automatically post-spec (configurable)
- [ ] Detects breaking changes in shared interfaces
- [ ] Generates integration tests for spec pairs
- [ ] Can be triggered manually: `dag integrate <spec-id>`
- [ ] Reports integration status in `dag status`

---

### 6. Git Merge Conflict Agent
**Problem:** Merge conflicts after worktree completion are painful.

**Goal:** Agentic flow that handles merge conflicts intelligently.

**Trigger Points:**
- After each spec commits to its worktree
- During layer staging branch merge
- During final merge to main/dev branch

**Capabilities:**
- Auto-resolve trivial conflicts (imports, formatting)
- AI-assisted conflict resolution for complex cases
- `--auto-resolve` flag for headless mode
- Escalation to user for uncertain conflicts
- Dry-run mode: show what conflicts would occur before merging

**Acceptance Criteria:**
- [ ] `dag merge --auto-resolve` attempts AI resolution
- [ ] `dag merge --dry-run` shows potential conflicts without applying
- [ ] Complex conflicts prompt user with context
- [ ] Conflict resolution logged and reviewable
- [ ] Rollback capability if merge goes wrong

---

### 8. User Observability Dashboard
**Problem:** Hard to understand DAG state at a glance — what's running, what's blocked, what's done.

**Goal:** Clear, actionable visibility into DAG execution.

**Ideas:**
- Rich `dag status` view: specs as a grid/timeline
- Live progress during `dag run`: which specs in parallel, ETA per spec
- Blocked reasons: "waiting on spec-001", "worktree conflict", "user review needed"
- Summary after completion: what succeeded, what failed, time per spec
- `dag watch` TUI mode for long-running DAGs
- Integration with notifications (desktop/push when your attention needed)

**Acceptance Criteria:**
- [ ] `dag status` shows visual timeline of spec execution
- [ ] Each spec shows: state icon, worktree path, time elapsed, block reason
- [ ] `dag run` shows live-updating progress (not just logs)
- [ ] Completion summary: table of specs with duration and result
- [ ] Blocked specs explain WHY they're blocked
- [ ] Option to export status as JSON for external dashboards

---

### 9. Issue Recovery & Troubleshooting
**Problem:** When DAG fails, hard to recover and resume without manual intervention.

**Goal:** Make common issues recoverable with clear guidance.

**Common Issues to Handle:**
1. **Spec fails mid-way** → Resume from failed spec, don't restart
2. **Worktree corrupted** → Detect, offer cleanup + retry
3. **Merge conflicts** → Clear conflict view, resolution options
4. **Spec stuck/running too long** → Timeout with option to cancel/retry
5. **Dependencies changed** → Detect, offer re-plan

**Ideas:**
- `dag doctor` command: diagnose common issues, suggest fixes
- `dag resume` alias: find interrupted DAG and continue
- `dag retry <spec-id>`: retry failed spec with options (clean, reuse, debug)
- `--debug` flag: verbose logging for troubleshooting
- On failure: show exactly what failed + next steps

**Acceptance Criteria:**
- [ ] `dag doctor` checks: worktree health, git state, spec completeness
- [ ] Failed specs can be retried individually: `dag retry 003-spec-name`
- [ ] Interrupt handling: graceful shutdown, can resume
- [ ] Error messages include: what failed, why, how to fix
- [ ] `--debug` shows: commands run, worktree paths, git state

---

### 7. Post-Spec Validation Agent (E2E Testing)
**Problem:** Specs enforce mock-only testing. Code looks good but may fail in production.

**Goal:** Real-world validation after each spec completes.

**Approaches to Evaluate:**
1. **Moltbot agent** — human-in-the-loop validation
2. **Claude Code automated** — headless e2e testing
3. **Hybrid** — automated with human escalation

**Validation Scope:**
- Real API calls (not mocks)
- Database writes/reads
- Side effects (webhooks, queues, etc.)
- Integration with external services

**Acceptance Criteria:**
- [ ] Define validation approach (Moltbot vs Claude vs Hybrid)
- [ ] Validation steps can be defined in dag.yaml
- [ ] Validation runs automatically post-spec (configurable)
- [ ] Failures block downstream specs or warn only (configurable)
- [ ] Validation results stored and viewable

See dstask #8 for planning task.

---

## Completed Tasks

_None yet_

---

## Priority Ranking (Draft)

**P0 (Blockers for DAG adoption):**
1. Pre-execution spec folder validation (Critical Requirements)
2. Issue Recovery & Troubleshooting (#9)

**P1 (High impact, daily use):**
3. User Observability Dashboard (#8)
4. Improve Parallel Execution UX (#1)
5. Spec Generation Workflow (#4)

**P2 (Important, can iterate):**
6. Improve Worktree Handling (#2)
7. Improve DAG YAML Schema (#3)
8. Git Merge Conflict Agent (#6)

**P3 (Advanced features):**
9. Post-Spec Integration Agent (#5)
10. Post-Spec Validation Agent (#7)

## Design Principles

1. **Observability First** — Users must always know what's happening and why
2. **Simple by Default** — Complex options available, but hidden until needed
3. **Fail Gracefully** — Clear error messages + recovery paths
4. **Consistent Naming** — All specs follow predictable folder scheme
5. **Pre-validate Everything** — Catch issues before execution starts

## Notes

- These tasks can be worked on in parallel (different specs)
- Some may depend on others (e.g., #5 and #6 need #4's spec generation)
- Consider creating autospec specs for complex features
- **Always ask:** "How does this make the user's life easier?"
