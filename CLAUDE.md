# dot-claude

Personal Claude Code plugin marketplace (`ben9`). Two plugins — `automake` (dev pipeline) and `claudetools` (authoring/maintenance tools) — plus a standalone `rules/` directory. See [docs/CODEBASE.md](docs/CODEBASE.md) for full architecture.

## Conventions

- **Each plugin is a self-contained top-level directory** (`automake/`, `claudetools/`) with its own `skills/` and/or `agents/` subdirectory. Do not share a `skills/`/`agents/` root across plugins: the `agents` field in `marketplace.json` has no exception for a shared marketplace-root source the way `skills` does, so a plugin declaring no agents would silently inherit every agent in a shared folder. Give each plugin its own directory instead.
- **Agent files must be flat**: `agents/<name>.md`, not `agents/<name>/AGENT.md`. Nested agent files are silently dropped by plugin loading (verified against Claude Code v2.1.228) — this is why `automake/agents/` is flat.
- **Skill files stay in the standard `<name>/SKILL.md` directory layout.**
- **`rules/` is not a plugin component.** The plugin schema has no `rules/` directory (only `skills/`, `commands/`, `agents/`, `hooks/`, `.mcp.json`, `.lsp.json`, `monitors/`, `bin/`, and a `settings.json` limited to `agent`/`subagentStatusLine`). Anything in `rules/` stays outside both plugins as standalone `.claude/rules/`-style config.

## Adding a skill or agent

Use `claudetools:create-skill` / `claudetools:create-agent` / `claudetools:create-rule` rather than hand-writing files — they scaffold from templates and ask the right placement questions.

When adding a new skill or agent to an existing plugin, no `marketplace.json` change is needed — each plugin entry's `source` points at the whole plugin directory, so new files under `skills/` or `agents/` are picked up automatically.

When adding a **new plugin**, add an entry to `.claude-plugin/marketplace.json` with `"source": "./<plugin-dir>"` and give it its own directory.

## Before committing changes to marketplace.json or plugin contents

```
claude plugin validate . --strict
```

To verify component counts (catches agent-bleed and path mistakes before they reach GitHub):

```
TESTHOME=$(mktemp -d) && HOME="$TESTHOME" claude plugin marketplace add . \
  && HOME="$TESTHOME" claude plugin install automake@ben9 \
  && HOME="$TESTHOME" claude plugin install claudetools@ben9 \
  && HOME="$TESTHOME" claude plugin details automake@ben9 \
  && HOME="$TESTHOME" claude plugin details claudetools@ben9 \
  && rm -rf "$TESTHOME"
```

Expect `automake` → 3 skills, 9 agents; `claudetools` → 5 skills, 0 agents.

## After a rename

Renaming the marketplace or a plugin doesn't migrate anyone's existing local registration. After merging a rename, re-register locally:

```
/plugin marketplace remove <old-name>
/plugin marketplace add wenbenz/dot-claude
```
