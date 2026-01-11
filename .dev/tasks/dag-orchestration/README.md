# DAG Multi-Spec Orchestration

Orchestrate parallel feature implementation across multiple specs using DAG-defined dependencies.

**[00-summary.md](00-summary.md)** - Full vision and concepts

---

## Current State

### Rename Completed: `dag` → `waves`

The task-level `dag` command has been renamed to `waves`. The `internal/dag/` package is now `internal/taskgraph/`.

| Command | Description |
|---------|-------------|
| `autospec waves [spec-name]` | Visualize task dependency waves within a single spec |
| `autospec implement --parallel` | Execute tasks concurrently by wave |

### What's Implemented (Task-Level - Spec 065)

- Task wave computation from `dependencies` field in `tasks.yaml`
- Parallel task execution within a single spec
- Per-task worktree isolation

### What's NOT Implemented (Spec-Level)

- No multi-spec orchestration (running multiple `autospec run` in parallel)
- No spec-level dependency tracking
- No unified status across multiple running specs
- No auto-merge of completed specs

---

## Planned Specs

| Spec | File | Scope |
|------|------|-------|
| 1 | [spec-1-schema-validation.md](spec-1-schema-validation.md) | DAG YAML schema, validation, visualization |
| 2 | [spec-2-run-sequential.md](spec-2-run-sequential.md) | Sequential multi-spec execution |
| 3 | [spec-3-parallel-status.md](spec-3-parallel-status.md) | Parallel execution, status command |
| 4 | [spec-4-resume-merge.md](spec-4-resume-merge.md) | Resume failed runs, merge automation |
| 5 | [spec-5-watch-monitor.md](spec-5-watch-monitor.md) | DAG watch & logs commands |
| 6 | [spec-6-worktree-copy-files.md](spec-6-worktree-copy-files.md) | Worktree setup & validation |
| 7 | [spec-7-dag-retry.md](spec-7-dag-retry.md) | Smart retry for failed specs |

---

## Recommended Order

1. **Spec 6** - Quick win, improves worktree UX
2. **Spec 1** - Foundation for spec-level orchestration
3. **Spec 2** - Core execution (sequential)
4. **Spec 3** - Add parallelization + status
5. **Spec 4** - Recovery and merge automation
6. **Spec 5** - Monitoring (nice-to-have)
7. **Spec 7** - Advanced recovery

---

## Summary: Implemented vs Planned

| Feature | Status | Spec |
|---------|--------|------|
| **Task-Level** | | |
| `autospec waves [spec]` | ✅ | 065, 085 |
| `autospec implement --parallel` | ✅ | 065 |
| **Spec-Level** | | |
| DAG YAML schema | ⬜ | 1 |
| `autospec dag validate/visualize` | ⬜ | 1 |
| `autospec dag run` (sequential) | ⬜ | 2 |
| `autospec dag run --parallel` | ⬜ | 3 |
| `autospec dag status` | ⬜ | 3 |
| `autospec dag resume/merge` | ⬜ | 4 |
| `autospec dag watch/logs` | ⬜ | 5 |
| `worktree.copy_dirs` config | ⬜ | 6 |
| `autospec dag retry` | ⬜ | 7 |
