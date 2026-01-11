# DAG App: Git Statistics CLI

A CLI tool for analyzing git repository statistics - author contributions, code churn, file hotspots, and generating reports. Shows high parallelism in the DAG.

## Why This is a Good DAG Example

- **High parallelism**: L1 has 4 specs that can all run simultaneously
- **Natural dependencies**: Reports genuinely need all stats collected first
- **Real utility**: Fills a gap between `git log` and heavy tools like GitPrime
- **Go ecosystem**: Uses go-git library, good for autospec/Go projects

## DAG File

Create `.autospec/dags/gitstats.yaml`:

```yaml
schema_version: "1.0"

dag:
  name: "GitStats CLI v1"

execution:
  max_parallel: 4
  timeout: "4h"
  base_branch: "main"
  on_conflict: "manual"

layers:
  - id: "L0"
    name: "Core Infrastructure"
    features:
      - id: "200-repo-reader"
        description: "Implement repository reader using go-git: open local repos, clone remote repos to temp dir, iterate commits with filters (date range, author, path), handle large repos with streaming"

      - id: "201-cli-setup"
        description: "Set up CLI with Cobra: 'gitstats' root command, global flags (--repo, --since, --until, --author, --format=table|json|csv), config file support, colored output"

  - id: "L1"
    name: "Statistics Collectors"
    depends_on: ["L0"]
    features:
      - id: "202-author-stats"
        description: "Collect author statistics: commits per author, lines added/removed, first/last commit dates, active days count; 'gitstats authors' command with sorting options"
        depends_on: ["200-repo-reader", "201-cli-setup"]

      - id: "203-file-stats"
        description: "Collect file statistics: change frequency per file, lines changed over time, file age, last modifier; 'gitstats files' command with --top=N flag"
        depends_on: ["200-repo-reader", "201-cli-setup"]

      - id: "204-commit-patterns"
        description: "Analyze commit patterns: commits by day of week, commits by hour, commit message length stats, conventional commit analysis; 'gitstats patterns' command"
        depends_on: ["200-repo-reader", "201-cli-setup"]

      - id: "205-churn-analysis"
        description: "Detect code churn: files changed frequently in short periods, 'hotspot' detection, files with high add+delete ratio; 'gitstats churn' command with threshold flags"
        depends_on: ["200-repo-reader", "201-cli-setup"]

  - id: "L2"
    name: "Visualizations"
    depends_on: ["L1"]
    features:
      - id: "206-timeline-viz"
        description: "ASCII timeline visualization: activity heatmap by week (like GitHub contribution graph), commit density over time; 'gitstats timeline' with --period=week|month|year"
        depends_on: ["202-author-stats", "204-commit-patterns"]

      - id: "207-tree-viz"
        description: "Directory tree visualization: show file tree with change counts, color by hotness, collapsible depth; 'gitstats tree' with --depth and --min-changes flags"
        depends_on: ["203-file-stats", "205-churn-analysis"]

  - id: "L3"
    name: "Reports"
    depends_on: ["L2"]
    features:
      - id: "208-html-report"
        description: "Generate HTML report: embed all stats and visualizations, interactive charts with Chart.js, single-file output; 'gitstats report --output=report.html'"
        depends_on: ["206-timeline-viz", "207-tree-viz"]

      - id: "209-pr-insights"
        description: "PR/merge analysis: identify long-lived branches, merge frequency, common merge conflicts by file; 'gitstats merges' command, works with main/master detection"
        depends_on: ["202-author-stats", "203-file-stats"]

  - id: "L4"
    name: "Integration"
    depends_on: ["L3"]
    features:
      - id: "210-ci-integration"
        description: "CI-friendly output: 'gitstats ci' outputs machine-readable JSON, exit codes for threshold violations (e.g., --max-churn=10), GitHub Actions example in README"
        depends_on: ["208-html-report"]
```

## Execution Order with Parallelism

```
L0:  [200-repo-reader] ──────┬──────> [201-cli-setup]
                             │
L1:    ┌─ [202-author-stats] ←──────┤   (4 specs run in parallel!)
       ├─ [203-file-stats] ←────────┤
       ├─ [204-commit-patterns] ←───┤
       └─ [205-churn-analysis] ←────┘
                │
L2:    [206-timeline-viz] ←─────────┤
       [207-tree-viz] ←─────────────┘
                │
L3:    [208-html-report] ←──────────┤
       [209-pr-insights] ←──────────┘
                │
L4:    [210-ci-integration]
```

**Parallelism Benefits**: L1 runs 4 specs simultaneously, dramatically reducing total wall-clock time vs sequential execution.

## Validation

```bash
# Initialize project
mkdir gitstats && cd gitstats
go mod init github.com/yourname/gitstats
autospec init

# Create constitution (important for Go projects)
autospec constitution

# Create the DAG file
mkdir -p .autospec/dags
# ... paste the YAML above into .autospec/dags/gitstats.yaml

# Validate the DAG structure
autospec dag validate .autospec/dags/gitstats.yaml

# Visualize the dependency graph
autospec dag visualize .autospec/dags/gitstats.yaml

# See what would run
autospec dag run .autospec/dags/gitstats.yaml --dry-run

# Run with parallelism (4 specs at once in L1)
autospec dag run .autospec/dags/gitstats.yaml --parallel --max-parallel 4

# Monitor progress
autospec dag watch  # In another terminal

# Check logs for a specific spec
autospec dag logs <run-id> 203-file-stats
```

## Expected Spec Sizes

| Spec | Est. Tasks | Files | Hours | Notes |
|------|-----------|-------|-------|-------|
| 200-repo-reader | 8-10 | 3-4 | 3-4 | go-git setup, streaming |
| 201-cli-setup | 5-7 | 3-4 | 2-3 | Cobra boilerplate |
| 202-author-stats | 6-8 | 2-3 | 2-3 | Aggregation logic |
| 203-file-stats | 6-8 | 2-3 | 2-3 | Path tracking |
| 204-commit-patterns | 5-7 | 2-3 | 2-3 | Time analysis |
| 205-churn-analysis | 6-8 | 2-3 | 3-4 | Algorithm complexity |
| 206-timeline-viz | 5-7 | 2-3 | 2-3 | ASCII art |
| 207-tree-viz | 6-8 | 2-3 | 2-3 | Tree rendering |
| 208-html-report | 8-12 | 4-5 | 4-5 | Template + assets |
| 209-pr-insights | 6-8 | 2-3 | 2-3 | Branch analysis |
| 210-ci-integration | 4-6 | 2-3 | 1-2 | Output formatting |

## Technical Notes

- Uses [go-git](https://github.com/go-git/go-git) for Git operations (pure Go, no git binary needed)
- Each spec description includes specific commands and flags
- Visualizations are ASCII-only (no external dependencies)
- HTML report embeds everything (single file output)

## Extensions (Future DAG)

These could be a follow-up DAG after v1:

- `211-github-integration`: Fetch PR data from GitHub API
- `212-diff-analysis`: Language-aware diff parsing
- `213-team-metrics`: Multi-author team analysis
- `214-trend-detection`: Identify improving/declining patterns
