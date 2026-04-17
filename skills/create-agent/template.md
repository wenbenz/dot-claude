---
name: agent-name
description: One sentence. What role does this agent play? What does it receive and return?
tools: Read, Glob, Grep
# effort: low | medium | high | max
# model: haiku | sonnet | opus
---

# Agent Name

One sentence describing this agent's single responsibility in the pipeline.

## Input

Describe what this agent receives:
- `$ARGUMENTS` — the primary input (e.g. path to spec doc, code content, test results)

## Output

Describe what this agent returns (its final message is captured by the caller):
- Format: (e.g. JSON, Markdown, plain text)
- Contents: (e.g. list of requirements, generated code, test plan)

## Steps

1. **Read input** — parse `$ARGUMENTS` or the provided content
2. **Core task** — describe the main work
3. **Return output** — format and emit the result

## Rules

- Stay focused on this agent's single role — do not do the next agent's job
- If input is missing or ambiguous, emit an error in the output format and stop
