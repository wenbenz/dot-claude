# dot-claude

Personal Claude Code plugin marketplace. Registered as **`ben9`**.

## Install

```
/plugin marketplace add wenbenz/dot-claude
/plugin install automake@ben9
/plugin install claudetools@ben9
```

## Plugins

### automake

Turns a spec or ticket into reviewed, tested, documented code with a pull request.

| Skill | Does |
|---|---|
| `/automake:dev-pipeline` | Runs the full pipeline: plan → code + tests → validate → review → document → open PR |
| `/automake:plan-tickets` | Turns raw input (issues, docs, ad-hoc text) into structured tickets |
| `/automake:create-pr` | Branch + worktree + push + open PR, standalone |

Backed by 8 pipeline agents (`planner`, `coder`, `test-writer`, `validator`, `reviewer`, `doc-patcher`, `pr-agent`, `ticket-analyst`) — see [automake/README.md](automake/README.md) for how they fit together.

### claudetools

Authoring and maintenance tools.

| Skill | Does |
|---|---|
| `/claudetools:create-agent` | Scaffold a new pipeline agent |
| `/claudetools:create-skill` | Scaffold a new skill |
| `/claudetools:create-rule` | Scaffold a new `.claude/rules/` file |
| `/claudetools:update-docs` | Sync `README.md` / `CLAUDE.md` / `docs/*.md` with the current code |
| `/claudetools:bungafy` | Compress a markdown file to cut token count |

## Other contents

`rules/french.md` — an unconditional rule (not part of either plugin) that corrects French grammar mistakes at the start of a response when the user writes in French.

## More

See [docs/CODEBASE.md](docs/CODEBASE.md) for architecture and how the pieces connect.
