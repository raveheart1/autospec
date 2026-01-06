# Skills vs Commands: Context Optimization

**Status:** Proposed
**Priority:** Medium
**Related:** `autospec init`, `internal/commands`, `internal/cli/admin`

---

## Problem Statement

Claude Code slash commands (`.claude/commands/*.md`) are **fully preloaded into context** at startup, consuming significant tokens regardless of whether they're invoked.

Current autospec command files:
```
autospec.implement:     12.8KB → 3.2k tokens
autospec.tasks:          9.5KB → 2.4k tokens
autospec.checklist:      9.4KB → 2.3k tokens
autospec.analyze:        9.4KB → 2.3k tokens
... (15 files total)
─────────────────────────────────────────────
Total:                  ~89KB  → ~21k tokens (10.5% of 200k context)
```

This is wasteful when users may only invoke 1-2 commands per session.

### How Slash Commands Load

1. **Startup**: Full content of all `.md` files loaded into system prompt
2. **SlashCommand tool metadata**: Names + descriptions (15k char budget)
3. **Result**: ~21k tokens consumed before user types anything

### How Skills Load (Better)

1. **Startup**: Only `name` + `description` from SKILL.md frontmatter loaded
2. **Activation**: When Claude decides to use skill, asks user for confirmation
3. **Full load**: Only after confirmation, full SKILL.md content enters context
4. **Result**: ~500 tokens at startup, full content only when needed

---

## Proposed Solution

### 1. Add Skill Installation Mode

Create skill versions of autospec commands that use lazy loading:

```
.claude/skills/autospec.specify/
├── SKILL.md           # Main content (loaded on activation)
├── references/        # Optional detailed docs
└── examples/          # Optional examples
```

**SKILL.md format:**
```markdown
---
name: autospec.specify
description: Generate YAML feature specification from natural language description
---

[Full prompt content here - only loaded when skill is activated]
```

### 2. Dual Installation Options

**Option A: Commands (current behavior)**
- Fast to invoke (`/autospec.specify`)
- Full context cost (~21k tokens)
- User explicitly triggers

**Option B: Skills (new)**
- Claude auto-suggests when relevant
- Minimal context cost (~500 tokens initially)
- Requires user confirmation to activate

### 3. Modified `autospec init` Flow

```
$ autospec init

Select installation mode for autospec prompts:
  [x] Skills (Recommended) - ~500 tokens, lazy-loaded
  [ ] Commands - ~21k tokens, always in context
  [ ] Both - Maximum flexibility, highest token cost

✓ Skills: 8 installed → .claude/skills/
✓ Config: created
```

### 4. New CLI Commands

```bash
# Install as skills (new)
autospec skills install [--target DIR]

# Install as commands (existing, deprecated path)
autospec commands install [--target DIR]

# Check what's installed
autospec skills check
autospec commands check
```

---

## Implementation Plan

### Phase 1: Skill Template Generation

**Files to create:**
- `internal/skills/templates.go` - Embedded skill templates
- `internal/skills/install.go` - Skill installation logic

**Template transformation:**
```go
// Convert command .md to skill directory structure
func ConvertCommandToSkill(cmdContent []byte) (*SkillTemplate, error) {
    // 1. Parse existing YAML frontmatter
    // 2. Create SKILL.md with name/description frontmatter
    // 3. Move full content below frontmatter
    return &SkillTemplate{
        Name:        "autospec.specify",
        Description: "Generate YAML feature specification",
        Content:     fullPromptContent,
    }, nil
}
```

**Directory structure:**
```
.claude/skills/
├── autospec.specify/
│   └── SKILL.md
├── autospec.plan/
│   └── SKILL.md
├── autospec.tasks/
│   └── SKILL.md
├── autospec.implement/
│   └── SKILL.md
├── autospec.clarify/
│   └── SKILL.md
├── autospec.analyze/
│   └── SKILL.md
├── autospec.checklist/
│   └── SKILL.md
└── autospec.constitution/
    └── SKILL.md
```

### Phase 2: CLI Commands

**Files to modify:**
- `internal/cli/admin/admin.go` - Add `skills` subcommand group
- `internal/cli/admin/skills_install.go` - New file
- `internal/cli/admin/skills_check.go` - New file

```go
// internal/cli/admin/skills_install.go
var skillsInstallCmd = &cobra.Command{
    Use:   "install",
    Short: "Install autospec skills (token-efficient, lazy-loaded)",
    Long: `Install autospec as Claude Code skills.

Skills use lazy loading - only name and description are loaded at startup.
Full content loads only when Claude activates the skill.

This saves ~20k tokens compared to slash commands.

Example:
  autospec skills install
  autospec skills install --target ./custom/skills`,
    RunE: runSkillsInstall,
}
```

### Phase 3: Init Integration

**Files to modify:**
- `internal/cli/config/init_cmd.go` - Add installation mode prompt

```go
func promptInstallMode(cmd *cobra.Command) string {
    options := []string{
        "skills (Recommended - ~500 tokens, lazy-loaded)",
        "commands (~21k tokens, always in context)",
        "both (Maximum flexibility)",
        "none (Manual installation later)",
    }
    // Return: "skills", "commands", "both", or "none"
}

func runInit(cmd *cobra.Command, args []string) error {
    // ... existing setup ...

    mode := promptInstallMode(cmd)

    switch mode {
    case "skills":
        skills.InstallTemplates(skillsDir)
    case "commands":
        commands.InstallTemplates(commandsDir)
    case "both":
        skills.InstallTemplates(skillsDir)
        commands.InstallTemplates(commandsDir)
    }

    // ... rest of init ...
}
```

### Phase 4: SKILL.md Template Format

Each skill needs proper frontmatter for Claude Code discovery:

```markdown
---
name: autospec.specify
description: Generate YAML feature specification from natural language description. Use when user wants to create a new spec for a feature.
---

# autospec.specify

You are a senior software architect...

[Full prompt content]
```

**Key frontmatter fields:**
- `name`: Skill identifier (required)
- `description`: Brief description for Claude's discovery (required, ~100 chars)
- Full content below frontmatter only loaded on activation

---

## Migration Path

### For Existing Users

```bash
# Check current installation
autospec commands check
# Output: 8 commands installed in .claude/commands/

# Migrate to skills
autospec skills install
autospec commands uninstall  # Optional: remove old commands

# Verify
autospec skills check
# Output: 8 skills installed in .claude/skills/
```

### Config Flag (Optional)

```yaml
# .autospec/config.yml
install_mode: skills  # or: commands, both
```

Remembered for future `autospec init` runs.

---

## Token Savings Analysis

| Mode | Startup Cost | Per-Invocation | Best For |
|------|--------------|----------------|----------|
| Commands | ~21k tokens | 0 (already loaded) | Heavy autospec users |
| Skills | ~500 tokens | ~2-3k per skill | Occasional users |
| Both | ~21.5k tokens | 0 | Maximum flexibility |

**Break-even point**: If you invoke <7 skills per session, skills mode saves tokens.

---

## Open Questions

1. **Should skills use the same names as commands?**
   - Same: `autospec.specify` - familiar, but may conflict
   - Different: `autospec-specify` or just `specify` - cleaner namespace

2. **Can commands and skills coexist?**
   - Need to test if Claude Code handles both gracefully
   - May cause duplicate suggestions

3. **Should we auto-migrate existing command users?**
   - Conservative: Prompt during init upgrade
   - Aggressive: Auto-migrate with backup

4. **What about non-Claude agents?**
   - Skills are Claude Code specific
   - Commands may be more portable to other agents (Cline, etc.)
   - Consider: Keep commands as "universal" format

---

## Files to Create/Modify

| File | Action |
|------|--------|
| `internal/skills/templates.go` | Create - Embedded skill templates |
| `internal/skills/install.go` | Create - Installation logic |
| `internal/skills/check.go` | Create - Check installed skills |
| `internal/cli/admin/skills.go` | Create - Skills command group |
| `internal/cli/admin/skills_install.go` | Create - Install subcommand |
| `internal/cli/admin/skills_check.go` | Create - Check subcommand |
| `internal/cli/config/init_cmd.go` | Modify - Add mode prompt |
| `internal/config/config.go` | Modify - Add `install_mode` field |

---

## Testing Strategy

### Unit Tests

1. **Template conversion:**
   - Command .md → SKILL.md directory structure
   - Frontmatter extraction and reformatting
   - Content preservation

2. **Installation:**
   - Skills installed to correct directory
   - Existing skills updated (not duplicated)
   - Non-autospec skills preserved

### Integration Tests

1. **Init flow:**
   - Mode prompt appears
   - Correct installation based on selection
   - Config persistence

2. **Claude Code compatibility:**
   - Skills discovered by Claude Code
   - Lazy loading works (verify via /context)
   - Activation confirmation appears

---

## References

- [Claude Code Skills Documentation](https://code.claude.com/docs/en/skills)
- [Slash Commands vs Skills](https://code.claude.com/docs/en/slash-commands)
- [Context7 Research on Loading Behavior](#) - See conversation context
- [init-agent-configuration.md](.dev/tasks/init-agent-configuration.md) - Related init improvements
