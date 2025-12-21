// Package workflow provides auto-commit instruction generation for agent prompt injection.
package workflow

// autoCommitInstructions contains the instructions injected into the agent prompt
// when auto-commit is enabled. These instructions guide the agent to create clean
// git commits following conventional commit format.
//
// The instructions are agent-agnostic and work with Claude, Gemini, and other
// supported agents. They cover:
// 1. .gitignore management for common ignorable patterns
// 2. Staging rules for appropriate files
// 3. Conventional commit message format
const autoCommitInstructions = `
## Auto-Commit Instructions

After completing your implementation work, create a clean git commit following these steps:

### Step 1: Update .gitignore

Before staging files, check for untracked files/folders that should be ignored.
Add common ignorable patterns to .gitignore if not already present:

**Dependency directories:**
- node_modules/
- vendor/
- .venv/, venv/
- __pycache__/

**Build outputs:**
- dist/, build/, out/
- target/
- bin/, obj/
- *.exe, *.dll, *.so, *.dylib

**IDE and editor files:**
- .idea/, .vscode/
- *.swp, *.swo
- .DS_Store, Thumbs.db

**Temporary and log files:**
- *.log
- *.tmp, *.temp
- .cache/, .tmp/

**Environment and secrets:**
- .env, .env.local, .env.*.local
- *.pem, *.key
- credentials.json

If .gitignore doesn't exist and ignorable files are present, create it with appropriate patterns.

### Step 2: Stage Appropriate Files

Stage all files that should be version controlled:

` + "```" + `bash
git add -A
` + "```" + `

**Do NOT stage these even if they exist:**
- Files matching .gitignore patterns
- Large binary files (>10MB) unless explicitly part of the project
- Generated files that should be rebuilt from source

**Verify staging with:**
` + "```" + `bash
git status
` + "```" + `

### Step 3: Create Conventional Commit

Create a commit message following conventional commit format:

` + "```" + `
type(scope): description

[optional body]
` + "```" + `

**Commit types:**
- feat: New feature
- fix: Bug fix
- docs: Documentation only
- style: Code style changes (formatting, semicolons, etc.)
- refactor: Code refactoring without feature changes
- test: Adding or updating tests
- chore: Maintenance tasks (dependencies, configs, etc.)

**Scope guidelines:**
- Determine scope based on the files/components you changed
- Use the most specific component name that covers the changes
- Examples: feat(auth), fix(api), docs(readme), refactor(config)

**Commit command:**
` + "```" + `bash
git commit -m "type(scope): brief description of changes"
` + "```" + `

### Important Notes

- If there are no changes to commit, skip the commit step
- If in a detached HEAD state, warn the user and skip the commit
- Focus on committing the work you just completed
- Keep commit messages concise but descriptive
`

// BuildAutoCommitInstructions returns the formatted auto-commit instructions
// for injection into the agent prompt. These instructions guide the agent
// to create a clean git commit after workflow completion.
func BuildAutoCommitInstructions() string {
	return autoCommitInstructions
}
