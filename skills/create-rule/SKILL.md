---
name: create-rule
description: Creates a new Claude Code rule from scratch. Use when the user asks to "create a rule", "add a rule", "make a rule", or wants persistent scoped instructions.
allowed-tools: Read Write Glob
argument-hint: [rule-name]
---

# Create Rule

1. **Name and purpose** — use `$ARGUMENTS` if provided; else ask: what to enforce, when to apply?
2. **Scope** — unconditional (no frontmatter) or path-scoped (`paths:` YAML list, e.g. `**/.claude/**/*.md`)
3. **Placement** — project-local (`.claude/rules/`) or global (`~/.claude/rules/`)
4. **Create file** — use [template.md](template.md); descriptive filename (e.g. `bungafy-after-edit.md`)
5. **Confirm** — show file and location; ask if adjustments needed
