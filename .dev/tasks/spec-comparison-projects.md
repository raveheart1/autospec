# Spec-Driven vs Prompt-Driven Development: Comparison Projects

10 mid-complexity project ideas to showcase why spec-driven development produces better results.

---

## Runner Script

**Location:** `~/.local/bin/spec-compare`

A bash script that runs spec-driven (autospec) and prompt-driven (claude -p) development side-by-side for comparison.

### What it does

1. Creates repos in `~/repos/<project>-spec` and `~/repos/<project>-prompt`
2. Initializes git with a README.md
3. **Spec-driven:** runs `autospec init` → `autospec constitution` → `autospec all`
4. **Prompt-driven:** runs `claude -p` with the same description (single shot, no iteration)
5. Logs output to `~/repos/<project>-{spec,prompt}.log`
6. Tracks timing (start, end, duration in HH:MM:SS)
7. Commits final state with `git add -A && git commit`

### Commands

```bash
# Run comparisons
spec-compare <1-10> spec      # Spec-driven only
spec-compare <1-10> prompt    # Prompt-driven only
spec-compare <1-10> both      # Both in parallel

# Management
spec-compare status           # Show running processes
spec-compare kill             # Kill all running processes
spec-compare clean <1-10>     # Remove repos/logs for project
spec-compare clean all        # Remove ALL repos/logs

# View logs
tail -f ~/repos/<project>-spec.log
tail -f ~/repos/<project>-prompt.log

# Check timing (at end of log)
tail -20 ~/repos/<project>-spec.log
```

### Notes

- Uses `ANTHROPIC_API_KEY=""` to ensure logged-in Claude session (no API charges)
- Runs processes in background with PID tracking
- All projects use Go with zero system dependencies

---

## 1. URL Shortener CLI

**What it does:** CLI tool that shortens URLs, stores mappings in a local JSON file, and provides stats.

**Why it showcases spec-driven:**
- Multiple commands (shorten, expand, stats, delete) need consistent patterns
- Edge cases: malformed URLs, duplicate handling, expiration logic
- Prompt-driven often forgets stats tracking or error handling consistency

**Complexity:** ~500 LOC | 4-5 files

<details>
<summary>Commands</summary>

```bash
# Spec-driven
spec-compare 1 spec

# Prompt-driven
spec-compare 1 prompt

# Both in parallel
spec-compare 1 both
```

</details>

---

## 2. Markdown Link Checker

**What it does:** Scans markdown files, validates all links (internal + external), reports broken ones.

**Why it showcases spec-driven:**
- Concurrent HTTP checking with rate limiting needs upfront design
- Multiple output formats (JSON, table, CI-friendly) require consistent structure
- Prompt-driven typically misses: relative path resolution, anchor validation, retry logic

**Complexity:** ~600 LOC | 5-6 files

<details>
<summary>Commands</summary>

```bash
# Spec-driven
spec-compare 2 spec

# Prompt-driven
spec-compare 2 prompt

# Both in parallel
spec-compare 2 both
```

</details>

---

## 3. Git Hooks Manager

**What it does:** Install/manage git hooks from a config file (like husky but simpler).

**Why it showcases spec-driven:**
- Hook lifecycle (install, uninstall, run, skip) has interdependencies
- Config schema needs validation rules defined upfront
- Prompt-driven often produces hooks that don't handle edge cases (no .git, nested repos)

**Complexity:** ~400 LOC | 4 files

<details>
<summary>Commands</summary>

```bash
# Spec-driven
spec-compare 3 spec

# Prompt-driven
spec-compare 3 prompt

# Both in parallel
spec-compare 3 both
```

</details>

---

## 4. Environment Variable Validator

**What it does:** Validates env vars against a schema file, generates .env.example, checks for secrets in code.

**Why it showcases spec-driven:**
- Type coercion rules (string→int→bool) need consistent specification
- Secret detection patterns require comprehensive regex upfront
- Prompt-driven misses: .env inheritance, CI mode, partial validation

**Complexity:** ~450 LOC | 5 files

<details>
<summary>Commands</summary>

```bash
# Spec-driven
spec-compare 4 spec

# Prompt-driven
spec-compare 4 prompt

# Both in parallel
spec-compare 4 both
```

</details>

---

## 5. API Mock Server

**What it does:** Reads OpenAPI spec, serves mock endpoints with realistic fake data.

**Why it showcases spec-driven:**
- Data generation rules per type need specification (emails, dates, IDs)
- Response delay simulation, error injection require defined behaviors
- Prompt-driven typically produces inconsistent fake data patterns

**Complexity:** ~700 LOC | 6-7 files

<details>
<summary>Commands</summary>

```bash
# Spec-driven
spec-compare 5 spec

# Prompt-driven
spec-compare 5 prompt

# Both in parallel
spec-compare 5 both
```

</details>

---

## 6. Cron Expression Parser Library

**What it does:** Parse cron expressions, calculate next N run times, validate expressions.

**Why it showcases spec-driven:**
- Edge cases are brutal: leap years, DST, month-end handling
- Spec forces you to define behavior for `*/15 * 31 2 *` upfront
- Prompt-driven almost always has subtle date math bugs

**Complexity:** ~500 LOC | 4 files

<details>
<summary>Commands</summary>

```bash
# Spec-driven
spec-compare 6 spec

# Prompt-driven
spec-compare 6 prompt

# Both in parallel
spec-compare 6 both
```

</details>

---

## 7. Config File Migrator

**What it does:** Migrate config files between formats (JSON↔YAML↔TOML) with schema versioning.

**Why it showcases spec-driven:**
- Version migration paths need explicit definition (v1→v2→v3 vs v1→v3)
- Comment preservation rules differ per format
- Prompt-driven produces inconsistent handling of nulls, empty arrays, nested objects

**Complexity:** ~550 LOC | 5-6 files

<details>
<summary>Commands</summary>

```bash
# Spec-driven
spec-compare 7 spec

# Prompt-driven
spec-compare 7 prompt

# Both in parallel
spec-compare 7 both
```

</details>

---

## 8. Changelog Generator

**What it does:** Parse conventional commits, generate CHANGELOG.md, support multiple output styles.

**Why it showcases spec-driven:**
- Commit categorization rules need exhaustive definition
- Breaking change detection requires clear patterns
- Prompt-driven often misses: scope handling, footer parsing, PR link injection

**Complexity:** ~500 LOC | 5 files

<details>
<summary>Commands</summary>

```bash
# Spec-driven
spec-compare 8 spec

# Prompt-driven
spec-compare 8 prompt

# Both in parallel
spec-compare 8 both
```

</details>

---

## 9. File Watcher with Actions

**What it does:** Watch directories, run commands on changes, with debouncing and filtering.

**Why it showcases spec-driven:**
- Debounce timing, ignore patterns, recursive watching need specification
- Action templating (pass filename, event type) requires defined syntax
- Prompt-driven typically has race conditions in rapid-change scenarios

**Complexity:** ~450 LOC | 4 files

<details>
<summary>Commands</summary>

```bash
# Spec-driven
spec-compare 9 spec

# Prompt-driven
spec-compare 9 prompt

# Both in parallel
spec-compare 9 both
```

</details>

---

## 10. License Compliance Checker

**What it does:** Scan dependencies, identify licenses, flag incompatibilities, generate NOTICE file.

**Why it showcases spec-driven:**
- License compatibility matrix needs explicit definition (MIT+GPL=?, Apache+BSD=?)
- SPDX expression parsing has edge cases
- Prompt-driven misses: transitive deps, dual-licensed packages, override rules

**Complexity:** ~600 LOC | 5-6 files

<details>
<summary>Commands</summary>

```bash
# Spec-driven
spec-compare 10 spec

# Prompt-driven
spec-compare 10 prompt

# Both in parallel
spec-compare 10 both
```

</details>

---

## Quick Reference

```bash
# Run any project comparison
spec-compare <1-10> <spec|prompt|both>

# Examples
spec-compare 6 both      # Cron parser - both approaches in parallel
spec-compare 2 spec      # Link checker - spec-driven only
spec-compare 10 prompt   # License checker - prompt-driven only

# View logs (created in ~/repos/)
tail -f ~/repos/cron-parser-spec.log
tail -f ~/repos/cron-parser-prompt.log

# Check status
spec-compare status
```

---

## Evaluation Criteria

When comparing spec-driven vs prompt-driven for each project, measure:

| Metric | What to Compare |
|--------|-----------------|
| **Consistency** | Are similar operations handled the same way? |
| **Edge Cases** | How many edge cases were missed initially? |
| **Iteration Count** | How many prompts to reach feature parity? |
| **Test Coverage** | Did tests cover the right scenarios? |
| **Refactor Needed** | How much restructuring after initial pass? |

---

## Best Candidates for Maximum Contrast

If limited to 3 projects, these show the starkest difference:

1. **#6 Cron Expression Parser** - Edge cases expose spec value immediately
2. **#10 License Compliance Checker** - Domain rules require upfront specification
3. **#5 API Mock Server** - Consistency across endpoints shows spec benefits

These projects have enough complexity to matter, but are small enough to build twice in reasonable time.
