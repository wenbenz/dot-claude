# Agent Reference

## Valid frontmatter fields

Same as skills — see `skills/create-skill/reference.md` for the full list.

Key fields for agents:

| Field | Value for agents |
|---|---|
| `context` | Always `fork` |
| `user-invocable` | Usually `false` |
| `effort` | `medium` or `high` for most pipeline agents |
| `allowed-tools` | Scope tightly — agents should only have what they need |
| `agent` | Optional: `Explore`, `Plan`, or `general-purpose` |

## Conventions for pipeline agents

### Input
Agents receive their input via `$ARGUMENTS`. The orchestrator (usually a skill or another agent) is responsible for passing the right content.

Prefer structured formats:
- JSON for structured data (requirements, test plans)
- Markdown for prose content (specs, code with context)
- File paths when the content is large (the agent reads it directly)

### Output
The agent's **final message** is captured by the caller. Structure it clearly:
- Always include a status: `OK` or `ERROR`
- Include the payload the next agent needs
- Use a consistent format per agent so the orchestrator can parse it

### Error handling
If input is malformed or a required file is missing:
1. Emit a structured error message
2. Do NOT guess or proceed with bad input
3. Stop immediately

## Pipeline agent roles

| Agent | Receives | Returns |
|---|---|---|
| `analyst` | Path to spec doc | Requirements list (Markdown) |
| `architect` | Requirements list | Module/interface design (Markdown) |
| `coder` | Architecture + requirements | Code files (written to disk, returns file list) |
| `test-designer` | Requirements + architecture | Test plan (Markdown) |
| `test-writer` | Test plan + code files | Test files (written to disk, returns file list) |
| `validator` | Test file list | Pass/fail report + failure details |
| `reviewer` | Code + test files | Review report (Markdown) |
| `pr-agent` | Review approval + branch name | PR URL |

## Repo-provided interface skills

Agents that generate code or tests should delegate style decisions to the repo's own skills:

```markdown
## Conventions
Read and follow the repo's skill before generating any code:
- Code style: invoke `/how-to-code` or read `.claude/skills/how-to-code/SKILL.md`
- Test style: invoke `/how-to-test` or read `.claude/skills/how-to-test/SKILL.md`
```
