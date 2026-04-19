---
name: update-docs
description: Scans the codebase and updates all documentation files. Use when the user asks to "update docs", "update documentation", "sync docs", "update the readme", or mentions docs are outdated.
allowed-tools: Read Write Edit Glob Grep
effort: high
---

# Update Docs

Scan repository, update all existing docs to reflect current code. Write in **English**.

## Steps

1. **Inventory docs** — find and read: `README.md`, `CLAUDE.md`, `docs/*.md`
2. **Explore codebase** — understand purpose, how to run, directory structure, key components, architecture, dependencies, recent changes
3. **Update each doc** — compare against actual code; update only outdated/incorrect sections; correct inaccuracies
4. **Create missing docs**: `docs/CODEBASE.md`, `README.md`, `CLAUDE.md`

## Format for docs/CODEBASE.md

```markdown
# Codebase Overview

## Purpose
One or two sentences.

## Architecture
High-level structure.

## Directory Structure
Brief description per top-level dir.

## Key Components
Main modules, responsibilities, interactions.

## Data Flow
How request/operation moves through system (skip if N/A).

## Dependencies
Most important external deps and why used.
```

Use **Mermaid diagrams** for architecture, data flow, entity relationships, sequences. Keep docs concise — new developer should understand after reading.
