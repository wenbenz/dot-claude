---
name: create-agent
description: Creates a new Claude Code pipeline agent. Use when the user asks to "create an agent", "add an agent", "make an agent", or wants a specialized role in an automation pipeline.
allowed-tools: Read Write Glob
argument-hint: [agent-name]
---

# Create Agent

Create Claude Code pipeline agent — isolated subagent called programmatically with single-purpose role.

## Agent vs Skill

| | Skill | Agent |
|---|---|---|
| Invocation | User types `/skill-name` or context-triggered | Programmatically by orchestrator |
| Isolation | Main context by default | Runtime-isolated |
| Scope | Broad workflow | Single focused role |
| Output | Shown to user | Returned to caller |
| User-invocable | Yes | Usually no |

## Steps

1. **Name and role** — use `$ARGUMENTS` if provided; else ask: what is single responsibility in pipeline?
2. **Clarify if needed** — inputs? outputs? tools needed? which pipeline? repo-provided skills?
3. **Placement** — project-local (`.claude/agents/`) if references project-specific tools/conventions; else global (`~/.claude/agents/`)
4. **Create files** — use [template.md](template.md); see [reference.md](reference.md); keep under 400 lines
5. **Confirm** — show file(s) and location; ask if adjustments needed

## Folder structure

```
agents/
└── my-agent/
    ├── AGENT.md        # Main instructions (required)
    └── reference.md    # Reference docs loaded when needed
```
