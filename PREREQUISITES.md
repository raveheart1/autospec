# Prerequisites

## Required Dependencies

### 1. Claude Code CLI (Required)

Claude Code CLI is essential - this entire repository automates SpecKit workflows using Claude Code commands and hooks.

**Installation:**
See: https://www.claude.com/product/claude-code

**Verify installation:**
```bash
claude --version
```

### 2. SpecKit (Required)

SpecKit must be installed using uv (the only supported installation method):

```bash
# Install SpecKit using uv
# See: https://github.com/github/spec-kit
uv tool install specify-cli --from git+https://github.com/github/spec-kit.git

# Verify installation
specify --version
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
- Homebrew recommended for installing dependencies

**Linux:**
- Git package from your distribution's package manager
- No additional requirements

## Verification

Verify all required tools are installed:

```bash
# Verify Claude Code CLI is accessible
claude --version

# Verify git is accessible
git --version

# Verify SpecKit is installed
specify --version

# Check all dependencies at once
specify check
```

## Optional Dependencies

- **jq**: JSON processor (install via package manager)

## Installing Missing Dependencies

### Claude Code CLI

**Required installation guide:**
See: https://www.claude.com/product/claude-code

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

### jq (Optional)

**Ubuntu/Debian:**
```bash
sudo apt-get install jq
```

**macOS:**
```bash
brew install jq
```

**Arch Linux:**
```bash
sudo pacman -S jq
```

## Troubleshooting

### "Command not found: specify"

You need to install SpecKit first using `uv`:

```bash
# Install uv if not already installed
# See: https://github.com/astral-sh/uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# Install SpecKit
uv tool install specify-cli --from git+https://github.com/github/spec-kit.git

# Verify installation
specify --version
```

See installation instructions: https://github.com/github/spec-kit

### "Command not found: jq"

Install jq using your package manager (see above).

### Git not found on Windows

Ensure Git is installed and `git.exe` is in your system PATH. Git Bash is recommended for best compatibility.
