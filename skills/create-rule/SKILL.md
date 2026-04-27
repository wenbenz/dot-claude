---
name: create-rule
description: Creates a new Claude Code rule from scratch. Use when the user asks to "create a rule", "add a rule", "make a rule", or wants persistent scoped instructions.
allowed-tools: Read Write Glob
argument-hint: [rule-name]
---

# Create Rule

Create rule following Claude Code conventions.

## Steps

1. **Name and purpose** — use `$ARGUMENTS` if provided; else ask: what to enforce, when to apply?
2. **Scope** — all files (no `paths`) or specific patterns (e.g. `**/.claude/**/*.md`)?
3. **Placement** — project-local (`.claude/rules/`) or global (`~/.claude/rules/`)
4. **Create file** — use [template.md](template.md); descriptive filename (e.g. `bungafy-after-edit.md`)
5. **Confirm** — show file and location; ask if adjustments needed

## Frontmatter

One optional field: `paths` (YAML list of globs). No other fields valid.

```markdown
---
paths:
  - "src/**/*.ts"
  - "lib/**/*.ts"
---
```

Omit frontmatter for unconditional rules.
