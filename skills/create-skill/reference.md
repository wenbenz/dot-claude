# Skill Reference

## Valid frontmatter fields

| Field | Notes |
|---|---|
| `name` | Lowercase, hyphens only, max 64 chars. Defaults to directory name. |
| `description` | Under 250 chars. Front-load key trigger phrases. |
| `allowed-tools` | Space-separated list (e.g. `Read Write Glob`) |
| `effort` | `low`, `medium`, `high`, or `max` |
| `context` | `fork` to run in a subagent |
| `agent` | Subagent type when `context: fork` (e.g. `Explore`) |
| `argument-hint` | Shown in autocomplete (e.g. `[filename]`) |
| `disable-model-invocation` | `true` to require manual `/name` invocation |
| `user-invocable` | `false` to hide from `/` menu |
| `paths` | Glob patterns that limit when this skill activates |
| `model` | Model override for this skill |
| `shell` | `bash` (default) or `powershell` for shell command blocks |
| `hooks` | Hooks scoped to this skill's lifecycle |

Do NOT use: `version`, or any field not in this list.

## String substitutions

| Variable | Description |
|---|---|
| `$ARGUMENTS` | All arguments passed when invoking the skill |
| `$ARGUMENTS[N]` | Specific argument by 0-based index |
| `$N` | Shorthand for `$ARGUMENTS[N]` (e.g. `$0`, `$1`) |
| `${CLAUDE_SESSION_ID}` | Current session ID |
| `${CLAUDE_SKILL_DIR}` | Directory containing the skill's `SKILL.md` |

## Dynamic context injection

Run shell commands whose output is injected before the skill is sent to Claude.

Inline:
```markdown
Current branch: !`git branch --show-current`
```

Multi-line:
````markdown
```!
git status --short
git log --oneline -5
```
````
