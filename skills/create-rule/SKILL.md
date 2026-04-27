---
name: create-rule
description: Creates a new Claude Code rule from scratch. Use when the user asks to "create a rule", "add a rule", "make a rule", or wants persistent scoped instructions.
allowed-tools: Read Write Glob
argument-hint: [rule-name]
---

# Create Rule

Create a new rule following Claude Code conventions.

## Steps

1. **Name and purpose** — use `$ARGUMENTS` if provided; else ask: what should the rule enforce, when should it apply?
2. **Scope** — should it apply to all files (`no paths`), or only specific patterns (e.g. `**/.claude/**/*.md`)? Ask if not obvious.
3. **Placement** — project-local (`.claude/rules/`) if project-specific; else global (`~/.claude/rules/`)
4. **Create file** — use [template.md](template.md); filename should be descriptive (e.g. `bungafy-after-edit.md`)
5. **Confirm** — show file and location; ask if adjustments needed

## Frontmatter

Only one optional field: `paths` (YAML list of glob patterns). No other frontmatter fields are valid for rules.

```markdown
---
paths:
  - "src/**/*.ts"
  - "lib/**/*.ts"
---
```

Omit the frontmatter block entirely for rules that apply unconditionally.
