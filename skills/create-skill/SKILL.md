---
name: create-skill
description: Creates a new Claude Code skill from scratch. Use when the user asks to "create a skill", "add a skill", "make a skill", or wants to automate a repeatable workflow.
allowed-tools: Read Write Glob
argument-hint: [skill-name]
---

# Create Skill

Create a new skill following Claude Code conventions.

## Steps

1. **Determine the skill name and purpose**
   - If `$ARGUMENTS` is provided, use it as the skill name
   - Otherwise ask the user: what should this skill do, and what phrases should trigger it?

2. **Ask clarifying questions if needed**
   - What tools will this skill need? (Read, Write, Edit, Glob, Grep, Bash, etc.)
   - Should Claude invoke it automatically, or only when the user types `/skill-name`?
   - Is this a quick task or a thorough one? (determines `effort`)
   - Does it need to run in isolation? (determines `context: fork`)

3. **Decide where to place the skill**

   Place the skill in `.claude/skills/` (project-local) if ANY of the following are true:
   - It references project-specific commands, scripts, or tooling
   - It encodes conventions specific to this codebase
   - It would not make sense outside of this project

   Otherwise place it in `~/.claude/skills/` (global).

4. **Create the skill files**
   - Use [template.md](template.md) as the base for `SKILL.md`
   - See [reference.md](reference.md) for valid frontmatter fields and substitutions
   - Only create supporting files (`reference.md`, `template.md`, `examples/`, `scripts/`) if needed
   - Keep `SKILL.md` under 500 lines — move reference material to separate files if needed

5. **Confirm** — show the created file(s) and location to the user and ask if anything needs adjusting.

## Skill folder structure

```
my-skill/
├── SKILL.md        # Main instructions and navigation (required)
├── reference.md    # Reference docs — loaded when needed
├── template.md     # Template for Claude to fill in — loaded when needed
├── examples/
│   └── sample.md   # Example outputs — loaded when needed
└── scripts/
    └── helper.sh   # Scripts — executed, not loaded
```
