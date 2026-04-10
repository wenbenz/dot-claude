---
name: create-agent
description: Create a new Claude Code sub-agent from scratch. Use when the user asks to "create an agent", "add a sub-agent", "make an agent", or wants to configure a specialized AI assistant for a repeatable task.
allowed-tools: Read Write Glob
effort: medium
argument-hint: [agent-name]
---

# Create Agent

Create a new Claude Code sub-agent with a custom system prompt, tool restrictions, and configuration.

## Steps

1. **Determine the agent name and purpose**
   - If `$ARGUMENTS` is provided, use it as the agent name
   - Otherwise ask: what should this agent do, and when should Claude delegate to it?

2. **Ask clarifying questions**
   - **Tools**: restrict with `tools:` (allowlist) or `disallowedTools:` (denylist)? Omit to inherit all tools.
   - **Model**: `haiku` (fast/cheap), `sonnet` (balanced), `opus` (complex), or `inherit` (default)
   - **Isolation**: `isolation: worktree` for an isolated git checkout?
   - **Memory**: `memory: user` (all projects), `project` (shareable via VCS), or `local` (not committed)?
   - **Background**: `background: true` to always run as a background task?
   - **Effort**: `low | medium | high | max` (default: inherits from session)
   - **Permission mode**: `default`, `acceptEdits`, `auto`, `dontAsk`, or `bypassPermissions`

3. **Decide where to place the agent**

   Place in `.claude/agents/` (project-local) if ANY of the following are true:
   - It references project-specific commands, scripts, or tooling
   - It encodes conventions specific to this codebase
   - It would not make sense outside of this project

   Otherwise place in `~/.claude/agents/` (global — available in all projects).

4. **Write a focused system prompt**
   - Describe what the agent does, not how Claude Code works
   - Be specific about expected output format if relevant
   - If `memory:` is set, include instructions for reading and updating the memory directory

5. **Create the agent file** using this format:

   ```markdown
   ---
   name: agent-name
   description: When Claude should delegate to this agent — be specific about trigger phrases
   tools: Read, Glob, Grep        # omit to inherit all tools from the parent session
   model: sonnet                  # omit to inherit from the parent conversation
   ---

   You are a specialized assistant for...
   ```

6. **Confirm** — show the created file and its location. Remind the user:
   > Agents are loaded at session start. Run `/agents` or restart the session to make it available immediately.

## Field reference

| Field             | Notes |
|:------------------|:------|
| `name`            | Lowercase, hyphens only. Must be unique. |
| `description`     | When Claude should delegate here — front-load key trigger phrases. |
| `tools`           | Allowlist. Comma-separated. Omit to inherit all tools. |
| `disallowedTools` | Denylist. Applied before `tools`. |
| `model`           | `haiku`, `sonnet`, `opus`, a full model ID, or `inherit`. |
| `permissionMode`  | `default`, `acceptEdits`, `auto`, `dontAsk`, `bypassPermissions`, `plan`. |
| `maxTurns`        | Max agentic turns before the agent stops. |
| `skills`          | Skills whose full content is injected at startup (not inherited from parent). |
| `mcpServers`      | MCP servers scoped to this agent only. |
| `hooks`           | Lifecycle hooks scoped to this agent. |
| `memory`          | `user`, `project`, or `local` — enables cross-session learning. |
| `background`      | `true` to always run as a background task. |
| `effort`          | `low`, `medium`, `high`, or `max`. |
| `isolation`       | `worktree` to run in a temporary isolated git checkout. |
| `color`           | `red`, `blue`, `green`, `yellow`, `purple`, `orange`, `pink`, `cyan`. |
| `initialPrompt`   | Auto-submitted as first user turn when agent runs as main session agent. |
