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
   - Any recent changes that may have made docs stale

3. **Update each doc** — for each file found in step 1:
   - Compare the current content against what the code actually does
   - **Only update sections that are outdated or incorrect** — preserve sections that are still accurate
   - Correct any inaccurate content even if it was written manually
   - If a doc references something that no longer exists, correct it

4. **Create missing docs** — create any of the following that don't exist:
   - `docs/CODEBASE.md` — architecture overview (see format below)
   - `README.md` — project description, setup, and usage
   - `CLAUDE.md` — project context and conventions for Claude

## Format for docs/CODEBASE.md

```markdown
# Codebase Overview

## Purpose
One or two sentences describing what this project does.

## Architecture
High-level description of how the system is structured.

## Directory Structure
Brief description of each top-level directory.

## Key Components
Main modules/packages, their responsibilities, and how they interact.

## Data Flow
How a request or operation moves through the system (skip if not applicable).

## Dependencies
Most important external dependencies and why they are used.
```

Use **Mermaid diagrams** where they add clarity — prefer them over text descriptions for:
- Architecture diagrams
- Data flow / request lifecycle
- Entity relationships
- Sequence diagrams

Keep all documentation concise — a new developer should understand the project after reading.
