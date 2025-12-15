# Prerequisites

## Required Dependencies

### 1. Claude Code CLI (Required)

Claude Code CLI is essential - autospec orchestrates feature development workflows using Claude Code commands.

**Installation:**
See: https://claude.ai/download

**Verify installation:**
```bash
claude --version
```

### 2. Git (Required)

Git is required for spec detection from branch names and general version control.

**Verify installation:**
```bash
git --version
```

### 3. Platform-Specific Requirements

**All Platforms:**
- Git must be installed and available in PATH

**Windows:**
- Git Bash recommended for best compatibility
- Ensure `git.exe` is in your system PATH
- PowerShell 5.0+ supported

**macOS:**
- Xcode Command Line Tools (includes git): `xcode-select --install`

**Linux:**
- Git package from your distribution's package manager

## Verification

Run the doctor command to verify all dependencies:

```bash
autospec doctor
```

Or verify manually:

```bash
# Verify Claude Code CLI
claude --version

# Verify Git
git --version
```

## Installing Missing Dependencies

### Claude Code CLI

**Installation guide:**
See: https://claude.ai/download

### Git

**Ubuntu/Debian:**
```bash
sudo apt-get install git
```

**macOS:**
```bash
xcode-select --install
```

**Windows:**
Download from https://git-scm.com/download/win

## Note on GitHub SpecKit

autospec was originally inspired by [GitHub SpecKit](https://github.com/github/spec-kit), but is now a **fully standalone tool**. The command templates in `internal/commands/templates/` were initially generated using SpecKit's tooling, but autospec now embeds and manages these commands independently.

**You do NOT need to install SpecKit** - autospec includes everything needed out of the box.
