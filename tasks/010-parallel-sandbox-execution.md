# 010 - Parallel Spec Execution with Sandbox Isolation

**Priority:** Medium
**Status:** Research Complete
**Created:** 2024-12-13

## Summary

Enable running multiple specs in parallel using git worktrees for file isolation and `srt` (sandbox-runtime) for process isolation.

## Tasks

- [ ] Add git worktree management to Go binary (`internal/worktree/`)
- [ ] Integrate `srt` CLI for sandbox isolation
- [ ] Add `autospec parallel <spec1> <spec2> ...` command
- [ ] Add `.srt-settings.json` default config generation
- [ ] Document sandbox network allowlist requirements
- [ ] Add `--sandbox` flag to implement command

## References

- Research: `docs/CLAUDE-AGENT-SDK-EVALUATION.md`
- Sandbox runtime: https://github.com/anthropic-experimental/sandbox-runtime
