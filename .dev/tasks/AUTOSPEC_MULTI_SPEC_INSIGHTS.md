# Autospec Multi-Spec Parallel Execution Insights

> Observations and recommendations from running 5+ parallel autospec workflows with GLM 4.7 free via opencode agent.

---

## Current Pain Points

### 1. No Native Multi-Worktree Orchestration
Autospec has `autospec worktree create/remove/list` commands but lacks:
- **Parallel execution manager** - no way to spawn N specs across N worktrees simultaneously
- **Unified status dashboard** - must manually `cd` into each worktree to check `autospec st`
- **Cross-worktree monitoring** - no single command to see all running specs

### 2. Manual Dependency Tracking
The AUTOSPEC_FEATURES.md file tracks dependencies manually:
```
| 004 | Instagram | 003 (shared auth) | [ ] |
| 051 | Retry | 050 | [ ] |
```
But autospec doesn't:
- Enforce dependency ordering
- Auto-schedule dependent specs after prerequisites complete
- Block specs until dependencies are merged

### 3. Inconsistent Failure Handling
Different failure modes require different recovery:
- **Specify failed** - restart from scratch
- **Plan failed** - can sometimes resume with `--resume`
- **Implement failed (incomplete tasks)** - can resume with `autospec implement --phase N`
- **Tests failed after implement** - needs manual intervention

---

## Proposed Commands for Multi-Spec DAG Execution

### `autospec dag init <dag-file.yaml>`
Define a directed acyclic graph of specs with dependencies:

```yaml
# dag.yaml
specs:
  050-partial-success:
    description: "Improve error handling"
    depends_on: []

  051-retry-backoff:
    description: "Retry with exponential backoff"
    depends_on: [050-partial-success]

  020-thread-support:
    description: "Thread/multi-post support"
    depends_on: []

  004-instagram:
    description: "Instagram strategy"
    depends_on: [003-facebook-pages]  # Shared auth

parallel_limit: 5
agent: opencode
model: opencode/glm-4.7-free
```

### `autospec dag run [--parallel N] [--agent NAME]`
Execute the DAG respecting dependencies:
- Creates worktrees for each spec
- Runs independent specs in parallel (up to N)
- Queues dependent specs until prerequisites complete
- Handles failures per configurable policy

### `autospec dag status`
Unified view across all worktrees:
```
DAG Execution Status
====================

Completed (3):
  ✓ 050-partial-success  [14/14 tasks, merged]
  ✓ 003-facebook-pages   [24/24 tasks, merged]
  ✓ 051-retry-backoff    [20/20 tasks, merged]

Running (2):
  ~ 004-instagram        [22/30 tasks, 73%] - depends: 003-facebook ✓
  ~ 020-thread-support   [18/22 tasks, 82%] - depends: none

Queued (1):
  ○ 021-video-upload     - waiting for: 020-thread-support

Failed (2):
  ✗ 006-tumblr          [plan stage] - schema validation errors
  ✗ 021-video-upload    [50% impl] - phase 3 incomplete tasks
```

### `autospec dag retry [spec-name] [--force]`
Retry a failed spec with smart cleanup:
- Removes old spec artifacts
- Creates fresh worktree
- Restarts from specify stage

### `autospec dag merge [spec-name] [--to branch]`
Merge completed spec to target branch:
- Runs tests first
- Handles merge conflicts
- Updates DAG status

---

## Monitoring Improvements

### `autospec watch [--interval 30s]`
Real-time monitoring of current spec:
```
Watching specs/062-instagram-strategy...

[22:45:01] Phase 7/9 - Task 22/30
[22:45:15] | Edit     src/strategies/instagram.js
[22:45:18] | Bash     npm test -- tests/strategies/instagram.test.js
[22:45:20] 51 passing (26ms)
[22:45:25] Task 23/30 started...
```

### `autospec watch-all [--worktrees]`
Monitor all active worktrees:
```
┌─────────────────────┬────────┬──────────────┬──────────────┐
│ Worktree            │ Status │ Progress     │ Last Update  │
├─────────────────────┼────────┼──────────────┼──────────────┤
│ 004-instagram       │ impl   │ 22/30 (73%)  │ 2s ago       │
│ 020-thread-support  │ impl   │ 18/22 (82%)  │ 15s ago      │
│ 051-retry-backoff   │ done   │ 20/20 (100%) │ merged       │
└─────────────────────┴────────┴──────────────┴──────────────┘
```

---

## Agent Selection & Model Configuration

### Current Approach (Manual)
Must manually set `opencode.json` in each worktree:
```json
{
  "model": "opencode/glm-4.7-free"
}
```

### Proposed: `autospec config set-agent`
```bash
# Set default agent for all new worktrees
autospec config set agent.default opencode
autospec config set agent.model opencode/glm-4.7-free

# Override per-spec
autospec run -spti --agent opencode --model opencode/glm-4.7-free
```

---

## Failure Recovery Patterns

### Pattern 1: Schema Validation Failures (Plan Stage)
```bash
# Current: Must restart entirely
autospec worktree remove 006-tumblr
autospec worktree create 006-tumblr --branch spec/006-tumblr-v2
cd ../006-tumblr && autospec run -spti --agent opencode -y "..."

# Proposed: autospec retry with cleanup
autospec dag retry 006-tumblr --clean
```

### Pattern 2: Incomplete Tasks (Implement Stage)
```bash
# Current: Resume from phase
autospec implement --phase 3 --resume

# Proposed: Auto-resume on failure
autospec run --auto-resume --max-retries 3
```

### Pattern 3: Test Failures After Implementation
```bash
# Current: Manual fix or re-run
npm test  # See what fails
# Edit code manually or restart

# Proposed: AI-assisted test fixing
autospec fix-tests --agent opencode
```

---

## Lessons Learned from This Session

### What Worked
1. **GLM 4.7 free** - Functional but 5-10x slower than paid models
2. **Parallel worktrees** - Good isolation, no cross-contamination
3. **`autospec st`** - Quick status checks work well
4. **Resume capability** - `--resume` helps recover from interruptions

### What Didn't Work
1. **No unified monitoring** - Had to check each worktree manually
2. **Inconsistent failures** - Different stages fail differently
3. **Model configuration** - Had to manually copy opencode.json
4. **Missing .opencode/command/** - Worktrees didn't get skill files

### Recommendations
1. Add `worktree.copy_files` config to auto-copy essential files
2. Create unified multi-spec status command
3. Add DAG-based execution for dependency ordering
4. Implement auto-retry with configurable policies
5. Add test-failure auto-fix mode

---

## Session Statistics

| Metric | Value |
|--------|-------|
| Specs Attempted | 5 |
| Completed & Merged | 1 (051-retry-backoff) |
| Build Errors | 1 (020-thread-support) |
| Failed | 2 (006-tumblr, 021-video-upload) |
| Still Running | 1 (004-instagram at 73%) |
| Agent Used | opencode with GLM 4.7 free |
| Monitoring Method | Manual `autospec st` every 2 min |

---

*Generated during multi-spec parallel execution session on 2026-01-10*
