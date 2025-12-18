# Suggested Fixes from Session Observations

Based on analysis of 29+ autospec-triggered Claude sessions documented in `observations.md`, these are concrete fixes to reduce token waste and improve efficiency.

---

## Priority Matrix

| Fix | Severity | Token Savings | Effort |
|-----|----------|---------------|--------|
| 1. File Reading Discipline | CRITICAL | 30-50K/session | Low |
| 2. Phase Context Metadata | HIGH | 5-15K/session | Medium |
| 3. Context Efficiency | HIGH | 10-20K/session | Low |
| 4. Sandbox Documentation | MEDIUM | 2-5K/session | Low |
| 5. Large File Handling | MEDIUM | 5-10K/session | Medium |
| 6. Test Infrastructure Caching | MEDIUM | Variable | Medium |

---

## Commands

### Fix 1: File Reading Discipline (CRITICAL)
```bash
autospec specify "$(cat .dev/tasks/fixes/fix-1-file-reading-discipline.md)"
```

### Fix 2: Phase Context Metadata (HIGH)
```bash
autospec specify "$(cat .dev/tasks/fixes/fix-2-phase-context-metadata.md)"
```

### Fix 3: Context Efficiency (HIGH)
```bash
autospec specify "$(cat .dev/tasks/fixes/fix-3-context-efficiency.md)"
```

### Fix 4: Sandbox Documentation (MEDIUM)
```bash
autospec specify "$(cat .dev/tasks/fixes/fix-4-sandbox-documentation.md)"
```

### Fix 5: Large File Handling (MEDIUM)
```bash
autospec specify "$(cat .dev/tasks/fixes/fix-5-large-file-handling.md)"
```

### Fix 6: Test Infrastructure Caching (MEDIUM)
```bash
autospec specify "$(cat .dev/tasks/fixes/fix-6-test-infrastructure-caching.md)"
```

---

## Implementation Order

1. **Fix 1** (implement.md file discipline) - Immediate, high impact, low effort
2. **Fix 3** (specify/plan context efficiency) - Quick template updates
3. **Fix 4** (sandbox documentation) - Documentation only
4. **Fix 2** (phase context metadata) - Requires Go code changes
5. **Fix 5** (large file hints) - Schema and template changes
6. **Fix 6** (notes.yaml caching) - New artifact type

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Duplicate file reads/session | 5-23 | ≤2 |
| Checklists checks (non-existent) | 10-49/session | 0-1 |
| Sandbox workarounds/session | 15 | 0 |
| Redundant artifact reads after phase context | 18-57 | 0 |
| Token waste/session | 30-50K | <10K |
| Test infrastructure rediscovery | Every phase | Phase 1 only |
