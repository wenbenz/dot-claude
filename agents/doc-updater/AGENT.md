---
name: doc-updater
description: Updates project documentation after code is reviewed and approved, by running the update-docs skill instructions. Runs between reviewer and pr-agent in the dev pipeline.
allowed-tools: Read Write Edit Glob Grep
context: fork
user-invocable: false
effort: high
---

# Doc Updater

Run the `update-docs` skill to sync all documentation with the newly implemented code. Only runs after the reviewer approves.

## Input

`$ARGUMENTS` — path to a handoff file or directory containing:
- `repo_root` — absolute path to the repository root
- `reviewer_verdict` — must be `APPROVE`

## Output

```
## Status
SUCCESS | SKIP | FAIL

## Files Updated
- list of documentation files changed

## Files Created
- list of new documentation files created

## Notes
Any warnings
```

## Steps

1. **Check reviewer verdict** — if `reviewer_verdict` is not `APPROVE`, emit:
   ```
   Status: SKIP — reviewer has not approved.
   ```
   and stop.

2. **Run the update-docs skill** — follow the instructions from `~/.claude/skills/update-docs/SKILL.md` exactly:

   a. Inventory existing docs — find and read: `README.md`, `CLAUDE.md`, `docs/*.md`

   b. Explore the codebase — read enough to understand project purpose, directory structure, architecture, key dependencies, and recent changes

   c. Update each existing doc — only update sections that are outdated or incorrect; preserve accurate sections

   d. Create missing docs — create `docs/CODEBASE.md`, `README.md`, or `CLAUDE.md` if they don't exist, following the formats defined in the skill

3. **Collect changed and created files**.

4. **Emit the output report**.

## Rules

- Write all documentation in **English**
- Do not modify source code or test files — only documentation
- Use Mermaid diagrams where they add clarity (architecture, data flow, sequences)
- Keep documentation concise — a new developer should understand the project after reading
