# Agent Reference

## Valid frontmatter fields for agents

| Field | Notes |
|---|---|
| `name` | Lowercase, hyphens only. Must be unique. |
| `description` | When Claude should delegate here — front-load key trigger phrases. |
| `tools` | Allowlist. **Comma-separated** (e.g. `Read, Glob, Grep`). Omit to inherit all tools. |
| `disallowedTools` | Denylist. Applied before `tools`. |
| `model` | `haiku`, `sonnet`, `opus`, a full model ID, or `inherit`. |
| `permissionMode` | `default`, `acceptEdits`, `auto`, `dontAsk`, `bypassPermissions`, `plan`. |
| `maxTurns` | Max agentic turns before the agent stops. |
| `skills` | Skills whose full content is injected at startup. |
| `mcpServers` | MCP servers scoped to this agent only. |
| `hooks` | Lifecycle hooks scoped to this agent. |
| `memory` | `user`, `project`, or `local` — enables cross-session learning. |
| `background` | `true` to always run as a background task. |
| `effort` | `low`, `medium`, `high`, or `max`. |
| `isolation` | `worktree` to run in a temporary isolated git checkout. |
| `color` | `red`, `blue`, `green`, `yellow`, `purple`, `orange`, `pink`, `cyan`. |
| `initialPrompt` | Auto-submitted as first user turn when agent starts. |

**Note:** Agent files use `tools:` (comma-separated), not `allowed-tools:` (which is a skill field). There is no `context: fork` or `user-invocable` for agents.

## Conventions for pipeline agents

### Input
Pipeline agents receive their input as a path to a JSON handoff file. The orchestrator writes this file before calling the agent. Pass the file path as `$ARGUMENTS`.

Handoff file format:
```json
{
  "requirements_file": ".pipeline/requirements.md",
  "architecture_file": ".pipeline/architecture.md",
  "repo_root": "/abs/path/to/repo"
}
```

### Output
The agent's **final message** is captured by the caller. Structure it clearly:
- Always include a status line: `## Status\nSUCCESS | FAIL | ERROR`
- Include the payload the next agent needs
- Write large outputs (reports, plans) to `.pipeline/<agent>_report.md` and reference the path

### Error handling
If input is malformed or a required file is missing:
1. Emit a structured error with `## Status\nERROR` and a description
2. Do NOT guess or proceed with bad input
3. Stop immediately

## Pipeline agent roles

| Agent | Handoff in | Writes to disk | Returns |
|---|---|---|---|
| `analyst` | Spec file path (direct arg) | — | Requirements Markdown |
| `architect` | `requirements_file` | — | Architecture Markdown |
| `coder` | `requirements_file`, `architecture_file`, `repo_root` | Source files | File list |
| `test-designer` | `requirements_file`, `architecture_file` | `test_plan.md` | Test plan Markdown |
| `test-writer` | `test_plan_file`, `code_files`, `repo_root` | Test files | File list + notes |
| `validator` | `test_files`, `repo_root`, `validator_notes` | `validator_report.md` | Pass/fail report |
| `reviewer` | `requirements_file`, `architecture_file`, `code_files`, `test_files`, `validator_report` | `reviewer_report.md` | Verdict + issues |
| `doc-updater` | `repo_root`, `reviewer_verdict` | Doc files | File list |
| `pr-agent` | All of the above + `branch_name` | — | PR URL |

## Repo-provided interface skills

Agents that generate code or tests should delegate style decisions to the repo's own skills:

```markdown
## Conventions
Check for a local skill first, then fall back to global:
1. `.claude/skills/how-to-code/SKILL.md` — project-local code conventions
2. `~/.claude/skills/how-to-code/SKILL.md` — global fallback
If neither exists, infer conventions from existing code in the repo.
```
