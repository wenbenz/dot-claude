---
name: agent-name
description: One sentence. Role, inputs, outputs.
tools: Read, Glob, Grep
# effort: low | medium | high | max
# model: haiku | sonnet | opus
---

# Agent Name

Single-responsibility sentence.

## Input

- `$ARGUMENTS` — primary input (e.g. path to spec, code, test results)

## Output

- Format: (e.g. JSON, Markdown, plain text)
- Contents: (e.g. requirements list, generated code, test plan)

## Steps

1. **Read input** — parse `$ARGUMENTS`
2. **Core task** — main work
3. **Return output** — format and emit

## Rules

- Stay focused on single role — don't do next agent's job
- Missing or ambiguous input → emit error and stop
