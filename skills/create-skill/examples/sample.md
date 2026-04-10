# Example: update-docs skill

A well-written skill for reference.

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

Scan this repository and update all existing documentation to reflect the current state of the code. Write all documentation in **English**.

## Steps

1. **Inventory existing docs** — find and read all of the following that exist:
   - `README.md`
   - `CLAUDE.md`
   - `docs/*.md`

2. **Explore the codebase** — read enough to understand:
   - Project purpose and how to run it
   - Directory structure and key components
   - Architecture and data flow
   - Important dependencies

3. **Update each doc** — for each file found in step 1:
   - Only update sections that are outdated or incorrect
   - Correct inaccurate content even if written manually

4. **Create missing docs** — create any that don't exist:
   - `docs/CODEBASE.md`, `README.md`, `CLAUDE.md`
```

**Why this works:**
- Description has clear trigger phrases under 250 chars
- `allowed-tools` lists exactly what's needed
- `effort: high` signals this is a thorough task
- Steps are concrete and actionable
