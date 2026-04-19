---
name: dev-pipeline
description: Transforms a technical specification document into reviewed, tested, and documented code with a pull request. Orchestrates planner → coder + test-writer → validator → reviewer → doc-patcher → pr-agent. Use when the user asks to "run the dev pipeline", "implement this spec", or "build from spec".
allowed-tools: Read Write Glob Agent
argument-hint: [path/to/spec.md] [branch-name]
effort: max
---

# Dev Pipeline

Orchestrate the full pipeline from technical specification to merged pull request.

```!
echo "Repo root: $(git rev-parse --show-toplevel 2>/dev/null || echo '(not a git repo)')"
echo "Branch:    $(git branch --show-current 2>/dev/null || echo '(unknown)')"
```

## Pipeline Overview

```
spec.md
  │
  ▼
[planner]          → .pipeline/plan.md (requirements + architecture)
  │
  ├─────────────────────────┐
  ▼                         ▼
[coder]            [test-writer]     ← both read plan.md
  │                         │
  └──────────┬──────────────┘
             ▼
        [validator] ──FAIL──→ [coder] or [test-writer] (max 2 rounds)
             │ PASS
             ▼
        [reviewer]  ──CHANGES──→ [coder] (max 1 retry)
             │ APPROVE
             ▼
        [doc-patcher]       → updated docs (if any)
             │
             ▼
        [pr-agent]          → PR URL + CI monitored
```

## Handoff convention

Before calling each agent, write a JSON handoff file to `.pipeline/`. The agent reads this file from the path passed as its argument. Each agent also writes its report to `.pipeline/<agent>_report.md` for use by downstream agents.

---

## Steps

### 0. Setup

1. **Parse arguments**:
   - `$ARGUMENTS[0]` — path to a spec document **or** an inline description (required)
   - `$ARGUMENTS[1]` — branch name (optional; default derived below)

2. **Determine spec source** — two modes:

   **File mode**: `$ARGUMENTS[0]` is a path to an existing file.
   - Verify the file exists and is readable. Stop if not.
   - `spec_file` = `$ARGUMENTS[0]`
   - Default branch name: `feat/<spec-filename-without-extension>`

   **Inline mode**: `$ARGUMENTS[0]` is not an existing file path (treat as a plain-text description).
   - Also applies when the pipeline is invoked from conversation context (no explicit argument) — use the user's request text as the description.
   - Write the description to `.pipeline/spec.md` (create `.pipeline/` first).
   - `spec_file` = `.pipeline/spec.md`
   - Default branch name: `feat/pipeline-<timestamp>` (e.g. `feat/pipeline-20260417`)

3. **Create a git worktree** for the branch:
   - Worktree path: `<repo_root>/../pipeline-worktree-<branch-name>`
   - Run: `git worktree add -b <branch_name> <worktree_path>`
   - All subsequent file writes (code, tests, docs, `.pipeline/`) happen inside `<worktree_path>`.
   - Use `<worktree_path>` as `repo_root` in every downstream handoff.

4. **Create** `.pipeline/` directory inside the worktree — all intermediate artifacts go here.

---

### 1. Planner

Call the `planner` agent with `spec_file` as its argument. Write its output to `.pipeline/plan.md`.

**Stop if** the planner emits an ERROR or lists more than 3 open questions — show the questions to the user and ask for clarification.

---

### 2. Coder + Test Writer (parallel)

Write `.pipeline/handoff_coder.json`:
```json
{
  "requirements_file": ".pipeline/plan.md",
  "architecture_file": ".pipeline/plan.md",
  "repo_root": "<repo_root>"
}
```

Write `.pipeline/handoff_test_writer.json`:
```json
{
  "requirements_file": ".pipeline/plan.md",
  "code_files": [],
  "repo_root": "<repo_root>"
}
```

Call `coder` with `.pipeline/handoff_coder.json` and `test-writer` with `.pipeline/handoff_test_writer.json` **simultaneously**. Wait for both.

- Save coder's returned file list as `code_files`.
- Save test-writer's returned file list as `test_files` and its notes as `validator_notes`.

---

### 3. Validator loop (max 2 rounds)

Write `.pipeline/handoff_validator.json`:
```json
{
  "test_files": <test_files list>,
  "repo_root": "<repo_root>",
  "validator_notes": "<notes from test-writer>"
}
```
Call `validator` with `.pipeline/handoff_validator.json`. Write its output to `.pipeline/validator_report.md`.

- **If PASS** → proceed to step 4.
- **If FAIL** → read the routing summary:
  - Routed to `coder` → add `failure_details` to `.pipeline/handoff_coder.json`, call `coder` again
  - Routed to `test-writer` → add `failure_details` to `.pipeline/handoff_test_writer.json`, call `test-writer` again
  - Routed to `analyst` → **stop and ask the user** to clarify the requirement
  - After fixes, re-run `validator`
- **After 2 failed rounds** → stop and show `.pipeline/validator_report.md` to the user.

---

### 4. Reviewer

Write `.pipeline/handoff_reviewer.json`:
```json
{
  "requirements_file": ".pipeline/plan.md",
  "architecture_file": ".pipeline/plan.md",
  "code_files": <code_files>,
  "test_files": <test_files>,
  "validator_report": ".pipeline/validator_report.md"
}
```
Call `reviewer` with `.pipeline/handoff_reviewer.json`. Write its output to `.pipeline/reviewer_report.md`.

- **If APPROVE** → proceed to step 5.
- **If REQUEST CHANGES** → send BLOCKING issues back to `coder` (add as `review_issues` in handoff), reset validator round counter, return to step 3. Maximum **1 retry** — stop and show the report if still failing.

---

### 5. Doc Patcher

Write `.pipeline/handoff_doc_patcher.json`:
```json
{
  "code_files": <code_files>,
  "repo_root": "<repo_root>"
}
```
Call `doc-patcher` with `.pipeline/handoff_doc_patcher.json`. Save its list of updated files as `doc_files`.

---

### 6. PR Agent

Write `.pipeline/handoff_pr_agent.json`:
```json
{
  "repo_root": "<worktree_path>",
  "branch_name": "<branch_name>",
  "worktree_path": "<worktree_path>",
  "code_files": <code_files>,
  "test_files": <test_files>,
  "doc_files": <doc_files>,
  "artifact_dir": ".pipeline",
  "requirements_file": ".pipeline/plan.md",
  "reviewer_report": ".pipeline/reviewer_report.md",
  "reviewer_verdict": "APPROVE"
}
```

Call `pr-agent` with `.pipeline/handoff_pr_agent.json`. The pr-agent handles: reading PR description content, cleanup of `.pipeline/`, commit, push, PR creation, and CI monitoring.

Report the PR URL to the user when done.

After the PR is created (or if the pipeline is stopped early due to an error), remove the worktree:
```
git worktree remove --force <worktree_path>
```

---

## Error Handling

| Situation | Action |
|---|---|
| File mode: spec file not found | Stop immediately, tell the user |
| Inline mode: no argument and no conversation context | Stop, ask the user what to build |
| Worktree creation fails | Stop, tell the user (branch may already exist) |
| Planner finds >3 open questions | Stop, show the questions to the user |
| Validator fails 2 times | Stop, show last report to the user |
| Reviewer requests changes twice | Stop, show review to the user |
| CI fails 3 times | Stop, show CI log to the user |
| Any agent emits ERROR | Stop, show the error to the user |

## Rules

- Never commit `.pipeline/` artifacts to the repo
- Never push to `main` or `master`
- Always run coder and test-writer in parallel (step 2)
- Show progress to the user after each major step
- Validator round counter resets per reviewer cycle, not globally
