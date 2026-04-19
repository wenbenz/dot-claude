# Example: update-docs skill

```
skills/update-docs/SKILL.md
```

```yaml
---
name: update-docs
description: Scans the codebase and updates all documentation files. Use when the user asks to "update docs", "update documentation", "sync docs", "update the readme", or mentions docs are outdated.
allowed-tools: Read Write Edit Glob Grep
effort: high
---

# Update Docs

Scan repository, update existing docs to reflect current code. Write in **English**.

## Steps

1. **Inventory** — find and read: `README.md`, `CLAUDE.md`, `docs/*.md`
2. **Explore codebase** — understand purpose, structure, architecture, dependencies
3. **Update each doc** — only outdated/incorrect sections; correct inaccuracies
4. **Create missing** — `docs/CODEBASE.md`, `README.md`, `CLAUDE.md`
```

**Why this works:**
- Trigger phrases front-loaded in description (under 250 chars)
- `allowed-tools` lists exactly what's needed
- `effort: high` signals thorough task
- Steps concrete and actionable
