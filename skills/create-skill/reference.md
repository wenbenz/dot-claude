# Skill Reference

## Valid frontmatter fields

| Field | Notes |
|---|---|
| `name` | Lowercase, hyphens only, max 64 chars. Defaults to directory name. |
| `description` | Under 250 chars. Front-load key trigger phrases. |
| `allowed-tools` | Space-separated (e.g. `Read Write Glob`) |
| `effort` | `low`, `medium`, `high`, or `max` |
| `context` | `fork` to run in subagent |
| `agent` | Subagent type when `context: fork` (e.g. `Explore`) |
| `argument-hint` | Shown in autocomplete (e.g. `[filename]`) |
| `disable-model-invocation` | `true` to require manual `/name` invocation |
| `user-invocable` | `false` to hide from `/` menu |
| `paths` | Glob patterns limiting when skill activates |
| `model` | Model override |
| `shell` | `bash` (default) or `powershell` |
| `hooks` | Hooks scoped to this skill's lifecycle |

Do NOT use: `version`, or any unlisted field.

## String substitutions

| Variable | Description |
|---|---|
| `$ARGUMENTS` | All arguments passed when invoking |
| `$ARGUMENTS[N]` | Argument by 0-based index |
| `$N` | Shorthand (e.g. `$0`, `$1`) |
| `${CLAUDE_SESSION_ID}` | Current session ID |
| `${CLAUDE_SKILL_DIR}` | Directory containing `SKILL.md` |

## Dynamic context injection

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
