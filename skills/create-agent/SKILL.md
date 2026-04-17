---
name: create-agent
description: Creates a new Claude Code pipeline agent. Use when the user asks to "create an agent", "add an agent", "make an agent", or wants a specialized role in an automation pipeline.
allowed-tools: Read Write Glob
argument-hint: [agent-name]
---

# Create Agent

Create a new Claude Code agent — an isolated subagent called programmatically by an orchestrator, with a focused single-purpose role.

## Difference: Agent vs Skill

| | Skill | Agent |
|---|---|---|
| Invocation | User types `/skill-name` or triggered by context | Called programmatically by an orchestrator |
| Isolation | Runs in main context by default | Isolated by the runtime (no `context: fork` field needed) |
| Scope | Broad workflow (e.g. "create a PR") | Single focused role (e.g. "analyze requirements") |
| Output | Shown to user | Returned to calling agent/skill |
| User-invocable | Yes | Usually no — set `description:` to control delegation |

## Steps

1. **Determine the agent name and role**
   - If `$ARGUMENTS` is provided, use it as the agent name
   - Otherwise ask: what is this agent's single responsibility in the pipeline?

2. **Ask clarifying questions if needed**
   - What inputs does this agent receive? (e.g. spec doc, code files, test results)
   - What does it output? (e.g. requirements list, code, test plan)
   - What tools does it need? (Read, Write, Edit, Glob, Grep, Bash, etc.)
   - Which pipeline does it belong to? (e.g. dev-pipeline, test-pipeline)
   - Does it use any repo-provided skills as an interface? (e.g. `how-to-code`, `how-to-test`)

3. **Decide where to place the agent**

   Place in `.claude/agents/` (project-local) if ANY of the following:
   - It references project-specific tools, commands, or conventions
   - It delegates to project-local skills (`how-to-code`, `how-to-test`, etc.)
   - It would not make sense outside this project

   Otherwise place in `~/.claude/agents/` (global).

4. **Create the agent files**
   - Use [template.md](template.md) as the base for `AGENT.md`
   - See [reference.md](reference.md) for valid frontmatter and conventions
   - Keep `AGENT.md` under 400 lines — move reference material to separate files

5. **Confirm** — show the created file(s) and location. Ask if anything needs adjusting.

## Agent folder structure

```
agents/
└── my-agent/
    ├── AGENT.md        # Main instructions (required)
    └── reference.md    # Reference docs loaded when needed
```
