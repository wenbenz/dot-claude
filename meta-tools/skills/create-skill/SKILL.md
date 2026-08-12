---
name: create-skill
description: Creates a new Claude Code skill from scratch. Use when the user asks to "create a skill", "add a skill", "make a skill", or wants to automate a repeatable workflow.
allowed-tools: Read Write Glob
argument-hint: [skill-name]
---

# Create Skill

Create new skill following Claude Code conventions.

## Steps

1. **Name and purpose** — use `$ARGUMENTS` if provided; else ask: what should skill do, what phrases trigger it?
2. **Clarify if needed** — tools needed? auto-trigger or manual `/skill-name`? effort level? needs isolation?
3. **Placement** — project-local (`.claude/skills/`) if project-specific; else global (`~/.claude/skills/`)
4. **Create files** — use [template.md](template.md); see [reference.md](reference.md); supporting files only if needed; keep under 500 lines
5. **Confirm** — show file(s) and location; ask if adjustments needed

## Folder structure

```
my-skill/
├── SKILL.md        # Main instructions (required)
├── reference.md    # Reference docs — loaded when needed
├── template.md     # Template — loaded when needed
├── examples/
│   └── sample.md   # Examples — loaded when needed
└── scripts/
    └── helper.sh   # Scripts — executed, not loaded
```
