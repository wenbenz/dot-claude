# Agent Reference

## Valid frontmatter fields

| Field | Notes |
|---|---|
| `name` | Lowercase, hyphens only. Must be unique. |
| `description` | When to delegate — front-load key trigger phrases. |
| `tools` | Allowlist. **Comma-separated**. Omit to inherit all. |
| `disallowedTools` | Denylist. Applied before `tools`. |
| `model` | `haiku`, `sonnet`, `opus`, full model ID, or `inherit`. |
| `permissionMode` | `default`, `acceptEdits`, `auto`, `dontAsk`, `bypassPermissions`, `plan`. |
| `maxTurns` | Max agentic turns before stop. |
| `skills` | Skills injected at startup. |
| `mcpServers` | MCP servers scoped to this agent. |
| `hooks` | Lifecycle hooks scoped to this agent. |
| `memory` | `user`, `project`, or `local`. |
| `background` | `true` to always run as background task. |
| `effort` | `low`, `medium`, `high`, or `max`. |
| `isolation` | `worktree` for isolated git checkout. |
| `color` | `red`, `blue`, `green`, `yellow`, `purple`, `orange`, `pink`, `cyan`. |
| `initialPrompt` | Auto-submitted as first user turn. |

**Note:** Agents use `tools:` (comma-separated), not `allowed-tools:`. No `context: fork` or `user-invocable` for agents.

## Pipeline conventions

**Input:** Orchestrator writes JSON handoff file; agent receives path as `$ARGUMENTS`.

```json
{
  "requirements_file": ".pipeline/requirements.md",
  "architecture_file": ".pipeline/architecture.md",
  "repo_root": "/abs/path/to/repo"
}
```

**Output:** Agent's final message captured by caller. Always include `## Status\nSUCCESS | FAIL | ERROR`. Write large outputs to `.pipeline/<agent>_report.md`.

**Error handling:** Malformed input → emit `## Status\nERROR` with description and stop.

## Pipeline roles

| Agent | Handoff in | Writes | Returns |
|---|---|---|---|
| `analyst` | Spec path | — | Requirements MD |
| `architect` | `requirements_file` | — | Architecture MD |
| `coder` | requirements, architecture, repo_root | Source files | File list |
| `test-designer` | requirements, architecture | `test_plan.md` | Test plan MD |
| `test-writer` | test_plan, code_files, repo_root | Test files | File list + notes |
| `validator` | test_files, repo_root, notes | `validator_report.md` | Pass/fail report |
| `reviewer` | requirements, architecture, code, tests, validator | `reviewer_report.md` | Verdict |
| `doc-updater` | repo_root, reviewer_verdict | Doc files | File list |
| `pr-agent` | All above + branch_name | — | PR URL |

## Repo-provided interface skills

Code/test agents should delegate style to repo skills:
1. `.claude/skills/how-to-code/SKILL.md` — project-local
2. `~/.claude/skills/how-to-code/SKILL.md` — global fallback

If neither exists, infer from existing code.
